// Package trie implements a dense Merkle Patricia Trie. See the documentation on [Trie] for details.
package trie

import (
	"fmt"
	"strings"

	"github.com/NethermindEth/juno/core/crypto"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/db"
	// Todo: Go.19 introduced math/bits library. Replace bits-and-blooms/bitset with the math/bits.
	"github.com/bits-and-blooms/bitset"
)

// Storage is the Persistent storage for the [Trie]
type Storage interface {
	Put(key *bitset.BitSet, value *Node) error
	Get(key *bitset.BitSet) (*Node, error)
	Delete(key *bitset.BitSet) error
}

// Trie is a dense Merkle Patricia Trie (i.e., all internal nodes have two children).
//
// This implementation allows for a "flat" storage by keying nodes on their path rather than
// their hash, resulting in O(1) accesses and O(log n) insertions.
//
// The state trie [specification] describes a sparse Merkle Trie.
// Note that this dense implementation results in an equivalent commitment.
//
// Terminology:
//   - path: represents the path as defined in the specification. Together with len,
//     represents a relative path from the current node to the node's nearest non-empty child.
//   - len: represents the len as defined in the specification. The number of bits to take
//     from the fixed-length path to reach the nearest non-empty child.
//   - key: represents the storage key for trie [Node]s. It is the full path to the node from the
//     root.
//
// [specification]: https://docs.starknet.io/documentation/develop/State/starknet-state/
type Trie struct {
	height  uint
	rootKey *bitset.BitSet
	storage Storage
}

func NewTrie(storage Storage, height uint, rootKey *bitset.BitSet) *Trie {
	// Todo: set max height to 251 and set max key value accordingly
	return &Trie{
		storage: storage,
		height:  height,
		rootKey: rootKey,
	}
}

// RunOnTempTrie creates an in-memory Trie of height `height` and runs `do` on that Trie
func RunOnTempTrie(height uint, do func(*Trie) error) error {
	db, err := db.NewInMemoryDb()
	if err != nil {
		return err
	}
	defer db.Close()

	txn := db.NewTransaction(true)
	defer txn.Discard()

	trieTxn := NewTrieBadgerTxn(txn, nil)
	return do(NewTrie(trieTxn, height, nil))
}

// FeltToBitSet Converts a key, given in felt, to a bitset which when followed on a [Trie],
// leads to the corresponding [Node]
func (t *Trie) FeltToBitSet(k *felt.Felt) *bitset.BitSet {
	regularK := k.ToRegular()
	return bitset.FromWithLength(t.height, regularK.Impl()[:])
}

// FindCommonKey finds the set of common MSB bits in two key bitsets.
func FindCommonKey(longerKey, shorterKey *bitset.BitSet) (*bitset.BitSet, bool) {
	divergentBit := uint(0)

	for divergentBit <= shorterKey.Len() &&
		longerKey.Test(longerKey.Len()-divergentBit) == shorterKey.Test(shorterKey.Len()-divergentBit) {
		divergentBit++
	}

	commonKey := shorterKey.Clone()
	for i := uint(0); i < shorterKey.Len()-divergentBit+1; i++ {
		commonKey.DeleteAt(0)
	}
	return commonKey, divergentBit == shorterKey.Len()+1
}

// Path returns the path as mentioned in the [specification] for commitment calculations.
// Path is suffix of key that diverges from parentKey. For example,
// for a key 0b1011 and parentKey 0b10, this function would return the Path object of 0b0.
//
// [specification]: https://docs.starknet.io/documentation/develop/State/starknet-state/
func Path(key, parentKey *bitset.BitSet) *bitset.BitSet {
	path := key.Clone()
	// drop parent key, and one more MSB since left/right relation already encodes that information
	if parentKey != nil {
		path.Shrink(path.Len() - parentKey.Len() - 1)
		path.DeleteAt(path.Len() - 1)
	}
	return path
}

// storageNode is the on-disk representation of a [Node],
// where key is the storage key and node is the value.
type storageNode struct {
	key  *bitset.BitSet
	node *Node
}

// nodesFromRoot enumerates the set of [Node] objects that need to be traversed from the root
// of the Trie to the node which is given by the key.
// The [storageNode]s are returned in descending order beginning with the root.
func (t *Trie) nodesFromRoot(key *bitset.BitSet) ([]storageNode, error) {
	var nodes []storageNode
	cur := t.rootKey
	for cur != nil {
		node, err := t.storage.Get(cur)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, storageNode{
			key:  cur,
			node: node,
		})

		_, subset := FindCommonKey(key, cur)
		if cur.Len() >= key.Len() || !subset {
			return nodes, nil
		}

		if key.Test(key.Len() - cur.Len() - 1) {
			cur = node.right
		} else {
			cur = node.left
		}
	}

	return nodes, nil
}

// Get the corresponding `value` for a `key`
func (t *Trie) Get(key *felt.Felt) (*felt.Felt, error) {
	value, err := t.storage.Get(t.FeltToBitSet(key))
	if err != nil {
		return nil, err
	}
	return value.value, nil
}

// Put updates the corresponding `value` for a `key`
func (t *Trie) Put(key *felt.Felt, value *felt.Felt) error {
	// Todo: check key is not bigger than max key value for a trie height.

	nodeKey := t.FeltToBitSet(key)
	node := &Node{
		value: value,
	}

	// empty trie, make new value root
	if t.rootKey == nil {
		if value.IsZero() {
			return nil // no-op
		}

		if err := t.propagateValues([]storageNode{
			{key: nodeKey, node: node},
		}); err != nil {
			return err
		}
		t.rootKey = nodeKey
		return nil
	}

	nodes, err := t.nodesFromRoot(nodeKey)
	if err != nil {
		return err
	}

	// Replace if key already exist
	sibling := &nodes[len(nodes)-1]
	if nodeKey.Equal(sibling.key) {
		sibling.node = node
		if value.IsZero() {
			if err = t.deleteLast(nodes); err != nil {
				return err
			}
		} else if err = t.propagateValues(nodes); err != nil {
			return err
		}
		return nil
	}

	// trying to insert 0 to a key that does not exist
	if value.IsZero() {
		return nil // no-op
	}

	commonKey, _ := FindCommonKey(nodeKey, sibling.key)
	newParent := &Node{
		value: new(felt.Felt),
	}
	if nodeKey.Test(nodeKey.Len() - commonKey.Len() - 1) {
		newParent.left, newParent.right = sibling.key, nodeKey
	} else {
		newParent.left, newParent.right = nodeKey, sibling.key
	}

	makeRoot := len(nodes) == 1
	if !makeRoot { // sibling has a parent
		siblingParent := &nodes[len(nodes)-2]

		// replace the link to our sibling with the new parent
		if siblingParent.node.left.Equal(sibling.key) {
			siblingParent.node.left = commonKey
		} else {
			siblingParent.node.right = commonKey
		}
	}

	// replace sibling with new parent
	nodes[len(nodes)-1] = storageNode{
		key: commonKey, node: newParent,
	}
	// add new node to steps
	nodes = append(nodes, storageNode{
		key: nodeKey, node: node,
	})

	// push commitment changes
	if err = t.propagateValues(nodes); err != nil {
		return err
	} else if makeRoot {
		t.rootKey = commonKey
	}
	return nil
}

// deleteLast deletes the last node in the given list and recalculates commitment
func (t *Trie) deleteLast(affectedNodes []storageNode) error {
	last := affectedNodes[len(affectedNodes)-1]
	if err := t.storage.Delete(last.key); err != nil {
		return err
	}

	if len(affectedNodes) == 1 { // deleted node was root
		t.rootKey = nil
	} else {
		// parent now has only a single child, so delete
		parent := affectedNodes[len(affectedNodes)-2]
		if err := t.storage.Delete(parent.key); err != nil {
			return err
		}

		var siblingKey *bitset.BitSet
		if parent.node.left.Equal(last.key) {
			siblingKey = parent.node.right
		} else {
			siblingKey = parent.node.left
		}

		if len(affectedNodes) == 2 { // sibling should become root
			t.rootKey = siblingKey
		} else { // sibling should link to grandparent (len(affectedNodes) > 2)
			grandParent := &affectedNodes[len(affectedNodes)-3]
			// replace link to parent with a link to sibling
			if grandParent.node.left.Equal(parent.key) {
				grandParent.node.left = siblingKey
			} else {
				grandParent.node.right = siblingKey
			}

			if sibling, err := t.storage.Get(siblingKey); err != nil {
				return err
			} else {
				// rebuild the list of affected nodes
				affectedNodes = affectedNodes[:len(affectedNodes)-2] // drop last and parent
				// add sibling
				affectedNodes = append(affectedNodes, storageNode{
					key:  siblingKey,
					node: sibling,
				})

				// finally recalculate commitment
				return t.propagateValues(affectedNodes)
			}
		}
	}

	return nil
}

// Recalculates [Trie] commitment by propagating `bottom` values as described in the [docs]
//
// [docs]: https://docs.starknet.io/documentation/develop/State/starknet-state/
func (t *Trie) propagateValues(affectedNodes []storageNode) error {
	for idx := len(affectedNodes) - 1; idx >= 0; idx-- {
		cur := affectedNodes[idx]

		if (cur.node.left == nil) != (cur.node.right == nil) {
			panic("should not happen")
		}

		if cur.node.left != nil || cur.node.right != nil {
			// todo: one of the children is already in affectedNodes, use that instead of fetching from storage
			left, err := t.storage.Get(cur.node.left)
			if err != nil {
				return err
			}

			right, err := t.storage.Get(cur.node.right)
			if err != nil {
				return err
			}

			leftPath := Path(cur.node.left, cur.key)
			rightPath := Path(cur.node.right, cur.key)

			cur.node.value = crypto.Pedersen(left.Hash(leftPath), right.Hash(rightPath))
		}

		if err := t.storage.Put(cur.key, cur.node); err != nil {
			return err
		}
	}

	return nil
}

// Root returns the commitment of a [Trie]
func (t *Trie) Root() (*felt.Felt, error) {
	if t.rootKey == nil {
		return new(felt.Felt), nil
	}

	root, err := t.storage.Get(t.rootKey)
	if err != nil {
		return nil, err
	}

	path := Path(t.rootKey, nil)
	return root.Hash(path), nil
}

// RootKey returns db key of the [Trie] root node
func (t *Trie) RootKey() *bitset.BitSet {
	return t.rootKey
}

func (t *Trie) Dump() {
	t.dump(0, nil)
}

// Try to print a [Trie] in a somewhat human-readable form
/*
Todo: create more meaningful representation of trie. In the current format string, storage is being
printed but the value that is printed is the bitset of the trie node this is different from the
storage of the trie. Also, consider renaming the function name to something other than dump.

The following can be printed:
- key (which represents the storage key)
- path (as specified in the documentation)
- len (as specified in the documentation)
- bottom (as specified in the documentation)

The spacing to represent the levels of the trie can remain the same.
*/
func (t *Trie) dump(level int, parentP *bitset.BitSet) {
	if t.rootKey == nil {
		fmt.Printf("%sEMPTY\n", strings.Repeat("\t", level))
		return
	}

	root, err := t.storage.Get(t.rootKey)
	path := Path(t.rootKey, parentP)
	fmt.Printf("%sstorage : \"%s\" %d spec: \"%s\" %d bottom: \"%s\" \n", strings.Repeat("\t", level), t.rootKey.String(), t.rootKey.Len(), path.String(), path.Len(), root.value.Text(16))
	if err != nil {
		return
	}
	(&Trie{
		rootKey: root.left,
		storage: t.storage,
	}).dump(level+1, t.rootKey)
	(&Trie{
		rootKey: root.right,
		storage: t.storage,
	}).dump(level+1, t.rootKey)
}
