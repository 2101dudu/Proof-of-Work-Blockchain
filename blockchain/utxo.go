package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"
	"slices"

	"github.com/dgraph-io/badger"
)

var (
	UTXOPrefix   = []byte("UTXOSet-")
	prefixLength = len(UTXOPrefix)
)

type UTXOSet struct {
	Blockchain *BlockChain // refenrece a Blockchain for its inclusion of a database pointer
}

// retrieve amount of tokens aswell as the transactions' IDs whose outputs concern the address' recipient
func (u *UTXOSet) FindSpendableOutputs(publicKeyHash []byte, amountToSend int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			var val []byte
			err := item.Value(func(v []byte) error {
				// this func with val would only be called if item.Value() encounters no error.
				val = slices.Clone(v)
				return nil
			})
			Handle(err)

			k = bytes.TrimPrefix(k, UTXOPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(val)

			for outIdx, out := range outs.Outputs {
				if out.isLockedWithKey(publicKeyHash) && accumulated < amountToSend {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}

		return nil
	})

	Handle(err)

	return accumulated, unspentOutputs
}

// locate the unspent transaction outputs (UTXOs)
func (u *UTXOSet) FindUTXO(publicKeyHash []byte) []TransactionOutput {
	var UTXOs []TransactionOutput

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			item := it.Item()

			var val []byte
			err := item.Value(func(v []byte) error {
				// this func with val would only be called if item.Value() encounters no error.
				val = slices.Clone(v)
				return nil
			})
			Handle(err)
			outs := DeserializeOutputs(val)
			for _, out := range outs.Outputs {
				if out.isLockedWithKey(publicKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	Handle(err)

	return UTXOs
}

func (u *UTXOSet) CountTransactions() int {
	db := u.Blockchain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			counter++
		}

		return nil
	})
	Handle(err)

	return counter
}

func (u *UTXOSet) Reindex() {
	db := u.Blockchain.Database

	u.deleteByPrefix(UTXOPrefix)

	UTXO := u.Blockchain.FindUnspentTransactions()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(UTXOPrefix, key...)

			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}
		return nil
	})
	Handle(err)
}

func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.isCoinbase() {
				for _, in := range tx.Inputs {
					updatedOutputs := TransactionOutputs{}
					inID := append(UTXOPrefix, in.ID...)
					item, err := txn.Get(inID)
					Handle(err)
					var val []byte
					err = item.Value(func(v []byte) error {
						// this func with val would only be called if item.Value() encounters no error.
						val = slices.Clone(v)
						return nil
					})
					Handle(err)

					outs := DeserializeOutputs(val)

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Output {
							updatedOutputs.Outputs = append(updatedOutputs.Outputs, out)
						}
					}

					if len(updatedOutputs.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							log.Panic(err)
						}
					} else {
						if err := txn.Set(inID, updatedOutputs.Serialize()); err != nil {
							log.Panic(err)
						}
					}
				}
			}
			newOutputs := TransactionOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(UTXOPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})

	Handle(err)
}

func (u *UTXOSet) deleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	// the number of keys to delete at a time
	collectionSize := 100000
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectionSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectionSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectionSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}
