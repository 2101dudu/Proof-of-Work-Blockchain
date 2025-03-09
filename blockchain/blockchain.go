package blockchain

import (
	"fmt"
	"slices"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// create a new instance of a blockchain with a genesis block
func CreateBlockChain() *BlockChain {
	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	Handle(err)

	// write to the db
	err = db.Update(func(txn *badger.Txn) error {
		// if we dont have an entry with key "lh" (last hash), we dont have a
		// blockchain and we need to create one.
		// otherwise, we need to get the last hash from that DB
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			genesisBlock := genesis()
			fmt.Println("Genesis block created")

			err = txn.Set(genesisBlock.Hash, genesisBlock.Serialize())
			Handle(err)
			err = txn.Set([]byte("lh"), genesisBlock.Hash)

			lastHash = genesisBlock.Hash

			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			Handle(err)

			err = item.Value(func(val []byte) error {
				// this func with val would only be called if item.Value() encounters no error.
				lastHash = slices.Clone(val)
				return nil
			})
			return err
		}
	})

	Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

// create and append a new bock to the list of existing blocks
func (chain *BlockChain) AddBlock(data []byte) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(val []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(val)
			return nil
		})

		return err
	})

	newBlock := createBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handle(err)
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

		err = item.Value(func(val []byte) error {
			blockData = slices.Clone(val)

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
