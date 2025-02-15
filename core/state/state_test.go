package state

import (
	"encoding/json"
	"testing"

	"github.com/NethermindEth/juno/clients"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/core/trie"
	"github.com/NethermindEth/juno/db"
	"github.com/bits-and-blooms/bitset"
	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/assert"
)

func TestState_PutNewContract(t *testing.T) {
	testDb := db.NewTestDb()
	state := NewState(testDb)

	addr, _ := new(felt.Felt).SetRandom()
	classHash, _ := new(felt.Felt).SetRandom()

	_, err := state.GetContractClass(addr)
	assert.EqualError(t, err, "Key not found")

	testDb.Update(func(txn *badger.Txn) error {
		assert.Equal(t, nil, state.putNewContract(addr, classHash, txn))
		assert.EqualError(t, state.putNewContract(addr, classHash, txn), "existing contract")
		return nil
	})

	got, err := state.GetContractClass(addr)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, classHash.Equal(got))
}

func TestState_Root(t *testing.T) {
	testDb := db.NewTestDb()

	state := NewState(testDb)

	key, _ := new(felt.Felt).SetRandom()
	value, _ := new(felt.Felt).SetRandom()

	var newRootPath *bitset.BitSet

	// add a value and update db
	if err := testDb.Update(func(txn *badger.Txn) error {
		storage, err := state.getStateStorage(txn)
		assert.Equal(t, nil, err)
		assert.Equal(t, nil, storage.Put(key, value))

		err = state.putStateStorage(storage, txn)
		assert.Equal(t, nil, err)
		newRootPath = storage.FeltToBitSet(key)

		return err
	}); err != nil {
		t.Error(err)
	}

	expectedRootNode := new(trie.Node)
	if err := expectedRootNode.UnmarshalBinary(value.Marshal()); err != nil {
		t.Error(err)
	}

	expectedRootNode.UnmarshalBinary(value.Marshal())
	expectedRoot := expectedRootNode.Hash(trie.Path(newRootPath, nil))

	actualRoot, err := state.Root()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, actualRoot.Equal(expectedRoot))
}

func TestUpdate(t *testing.T) {
	updateJson := []byte(`{
  "block_hash": "0x47c3637b57c2b079b93c61539950c17e868a28f46cdef28f88521067f21e943",
  "new_root": "021870ba80540e7831fb21c591ee93481f5ae1bb71ff85a86ddd465be4eddee6",
  "old_root": "0000000000000000000000000000000000000000000000000000000000000000",
  "state_diff": {
    "storage_diffs": {
      "0x20cfa74ee3564b4cd5435cdace0f9c4d43b939620e4a0bb5076105df0a626c6": [
        {
          "key": "0x5",
          "value": "0x22b"
        },
        {
          "key": "0x313ad57fdf765addc71329abf8d74ac2bce6d46da8c2b9b82255a5076620300",
          "value": "0x4e7e989d58a17cd279eca440c5eaa829efb6f9967aaad89022acbe644c39b36"
        },
        {
          "key": "0x313ad57fdf765addc71329abf8d74ac2bce6d46da8c2b9b82255a5076620301",
          "value": "0x453ae0c9610197b18b13645c44d3d0a407083d96562e8752aab3fab616cecb0"
        },
        {
          "key": "0x5aee31408163292105d875070f98cb48275b8c87e80380b78d30647e05854d5",
          "value": "0x7e5"
        },
        {
          "key": "0x6cf6c2f36d36b08e591e4489e92ca882bb67b9c39a3afccf011972a8de467f0",
          "value": "0x7ab344d88124307c07b56f6c59c12f4543e9c96398727854a322dea82c73240"
        }
      ],
      "0x31c887d82502ceb218c06ebb46198da3f7b92864a8223746bc836dda3e34b52": [
        {
          "key": "0xdf28e613c065616a2e79ca72f9c1908e17b8c913972a9993da77588dc9cae9",
          "value": "0x1432126ac23c7028200e443169c2286f99cdb5a7bf22e607bcd724efa059040"
        },
        {
          "key": "0x5f750dc13ed239fa6fc43ff6e10ae9125a33bd05ec034fc3bb4dd168df3505f",
          "value": "0x7c7"
        }
      ],
      "0x31c9cdb9b00cb35cf31c05855c0ec3ecf6f7952a1ce6e3c53c3455fcd75a280": [
        {
          "key": "0x5",
          "value": "0x65"
        },
        {
          "key": "0xcfc2e2866fd08bfb4ac73b70e0c136e326ae18fc797a2c090c8811c695577e",
          "value": "0x5f1dd5a5aef88e0498eeca4e7b2ea0fa7110608c11531278742f0b5499af4b3"
        },
        {
          "key": "0x5aee31408163292105d875070f98cb48275b8c87e80380b78d30647e05854d5",
          "value": "0x7c7"
        },
        {
          "key": "0x5fac6815fddf6af1ca5e592359862ede14f171e1544fd9e792288164097c35d",
          "value": "0x299e2f4b5a873e95e65eb03d31e532ea2cde43b498b50cd3161145db5542a5"
        },
        {
          "key": "0x5fac6815fddf6af1ca5e592359862ede14f171e1544fd9e792288164097c35e",
          "value": "0x3d6897cf23da3bf4fd35cc7a43ccaf7c5eaf8f7c5b9031ac9b09a929204175f"
        }
      ],
      "0x6ee3440b08a9c805305449ec7f7003f27e9f7e287b83610952ec36bdc5a6bae": [
        {
          "key": "0x1e2cd4b3588e8f6f9c4e89fb0e293bf92018c96d7a93ee367d29a284223b6ff",
          "value": "0x71d1e9d188c784a0bde95c1d508877a0d93e9102b37213d1e13f3ebc54a7751"
        },
        {
          "key": "0x449908c349e90f81ab13042b1e49dc251eb6e3e51092d9a40f86859f7f415b0",
          "value": "0x6cb6104279e754967a721b52bcf5be525fdc11fa6db6ef5c3a4db832acf7804"
        },
        {
          "key": "0x48cba68d4e86764105adcdcf641ab67b581a55a4f367203647549c8bf1feea2",
          "value": "0x362d24a3b030998ac75e838955dfee19ec5b6eceb235b9bfbeccf51b6304d0b"
        },
        {
          "key": "0x5bdaf1d47b176bfcd1114809af85a46b9c4376e87e361d86536f0288a284b65",
          "value": "0x28dff6722aa73281b2cf84cac09950b71fa90512db294d2042119abdd9f4b87"
        },
        {
          "key": "0x5bdaf1d47b176bfcd1114809af85a46b9c4376e87e361d86536f0288a284b66",
          "value": "0x57a8f8a019ccab5bfc6ff86c96b1392257abb8d5d110c01d326b94247af161c"
        },
        {
          "key": "0x5f750dc13ed239fa6fc43ff6e10ae9125a33bd05ec034fc3bb4dd168df3505f",
          "value": "0x7e5"
        }
      ],
      "0x735596016a37ee972c42adef6a3cf628c19bb3794369c65d2c82ba034aecf2c": [
        {
          "key": "0x5",
          "value": "0x64"
        },
        {
          "key": "0x2f50710449a06a9fa789b3c029a63bd0b1f722f46505828a9f815cf91b31d8",
          "value": "0x2a222e62eabe91abdb6838fa8b267ffe81a6eb575f61e96ec9aa4460c0925a2"
        }
      ]
    },
    "nonces": {},
    "deployed_contracts": [
      {
        "address": "0x20cfa74ee3564b4cd5435cdace0f9c4d43b939620e4a0bb5076105df0a626c6",
        "class_hash": "0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8"
      },
      {
        "address": "0x31c887d82502ceb218c06ebb46198da3f7b92864a8223746bc836dda3e34b52",
        "class_hash": "0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8"
      },
      {
        "address": "0x31c9cdb9b00cb35cf31c05855c0ec3ecf6f7952a1ce6e3c53c3455fcd75a280",
        "class_hash": "0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8"
      },
      {
        "address": "0x6ee3440b08a9c805305449ec7f7003f27e9f7e287b83610952ec36bdc5a6bae",
        "class_hash": "0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8"
      },
      {
        "address": "0x735596016a37ee972c42adef6a3cf628c19bb3794369c65d2c82ba034aecf2c",
        "class_hash": "0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8"
      }
    ],
    "declared_contracts": []
  }
}`)

	var gatewayUpdate clients.StateUpdate
	err := json.Unmarshal(updateJson, &gatewayUpdate)
	if err != nil {
		t.Error(err)
	}

	coreUpdate := new(core.StateUpdate)
	coreUpdate.BlockHash = gatewayUpdate.BlockHash
	coreUpdate.NewRoot = gatewayUpdate.NewRoot
	coreUpdate.OldRoot = gatewayUpdate.OldRoot
	coreUpdate.StateDiff = new(core.StateDiff)
	for _, contract := range gatewayUpdate.StateDiff.DeployedContracts {
		coreUpdate.StateDiff.DeployedContracts = append(coreUpdate.StateDiff.DeployedContracts, struct {
			Address   *felt.Felt
			ClassHash *felt.Felt
		}{Address: contract.Address, ClassHash: contract.ClassHash})
	}

	coreUpdate.StateDiff.StorageDiffs = make(map[felt.Felt][]core.StorageDiff)
	for addrStr, diffs := range gatewayUpdate.StateDiff.StorageDiffs {
		addr, _ := new(felt.Felt).SetString(addrStr)
		for _, diff := range diffs {
			coreUpdate.StateDiff.StorageDiffs[*addr] = append(coreUpdate.StateDiff.StorageDiffs[*addr], core.StorageDiff{
				Key:   diff.Key,
				Value: diff.Value,
			})
		}
	}

	testDb := db.NewTestDb()
	state := NewState(testDb)

	assert.Equal(t, nil, state.Update(coreUpdate))
}

func TestUpdateNonce(t *testing.T) {
	coreUpdate := new(core.StateUpdate)
	coreUpdate.OldRoot = new(felt.Felt)
	coreUpdate.NewRoot, _ = new(felt.Felt).SetString("0x4bdef7bf8b81a868aeab4b48ef952415fe105ab479e2f7bc671c92173542368")
	addr, _ := new(felt.Felt).SetString("0x20cfa74ee3564b4cd5435cdace0f9c4d43b939620e4a0bb5076105df0a626c6")
	classHash, _ := new(felt.Felt).SetString("0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8")

	coreUpdate.StateDiff = new(core.StateDiff)
	coreUpdate.StateDiff.DeployedContracts = []core.DeployedContract{
		{
			Address: addr, ClassHash: classHash,
		},
	}
	testDb := db.NewTestDb()
	state := NewState(testDb)

	assert.NoError(t, state.Update(coreUpdate))

	nonce, err := state.GetContractNonce(addr)
	assert.NoError(t, err)
	assert.Equal(t, true, nonce.Equal(&felt.Zero))

	coreUpdate = new(core.StateUpdate)
	coreUpdate.OldRoot, _ = new(felt.Felt).SetString("0x4bdef7bf8b81a868aeab4b48ef952415fe105ab479e2f7bc671c92173542368")
	coreUpdate.NewRoot, _ = new(felt.Felt).SetString("0x6210642ffd49f64617fc9e5c0bbe53a6a92769e2996eb312a42d2bdb7f2afc1")
	coreUpdate.StateDiff = new(core.StateDiff)
	coreUpdate.StateDiff.Nonces = make(map[felt.Felt]*felt.Felt)

	nonce.SetUint64(1)
	coreUpdate.StateDiff.Nonces[*addr] = nonce
	assert.NoError(t, state.Update(coreUpdate))

	newNonce, err := state.GetContractNonce(addr)
	assert.NoError(t, err)
	assert.Equal(t, true, nonce.Equal(newNonce))
}
