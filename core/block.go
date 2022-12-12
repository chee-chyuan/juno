package core

import (
	"github.com/consensys/gnark-crypto/ecc/stark-curve/fp"
)

type Block struct {
	// The hash of this block’s parent
	ParentHash *fp.Element
	// The number (height) of this block
	Number uint64
	// The state commitment after this block
	GlobalStateRoot *fp.Element
	// The StarkNet address of the sequencer who created this block
	SequencerAddress *fp.Element
	// The time the sequencer created this block before executing transactions
	Timestamp uint64
	// The number of transactions in a block
	TransactionCount uint64
	// A commitment to the transactions included in the block
	TransactionCommitment *fp.Element
	// The number of events
	EventCount uint64
	// A commitment to the events produced in this block
	EventCommitment *fp.Element
	// The version of the StarkNet protocol used when creating this block
	ProtocolVersion uint64
	// Extraneous data that might be useful for running transactions
	ExtraData *fp.Element
}

func (b *Block) Hash() *fp.Element {
	// Todo: implement pedersen hash as defined here
	// https://docs.starknet.io/documentation/develop/Blocks/header/#block_hash
	return nil
}
