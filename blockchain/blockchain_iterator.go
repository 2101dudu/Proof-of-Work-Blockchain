package blockchain

import (
	"slices"

	"github.com/dgraph-io/badger"
)

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

// this iterator's Next() method traverses the linked list backwards
func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		var blockData []byte

		err = item.Value(func(v []byte) error {
			blockData = slices.Clone(v)

			return nil
		})

		block = Deserialize(blockData)

		return err
	})
	Handle(err)

	// point the current hash to the previous node, effectively traversing the list backwards
	iter.CurrentHash = block.PrevHash

	return block
}
