package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"slices"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// check if MANIFEST file exists, i.e., the DB
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// read and continue from an existing blockchain
func ContinueBlockChain() *BlockChain {
	var lastHash []byte

	if !DBexists() {
		fmt.Println("BlockChain has not been created yet. Create one!")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(val []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(val)
			return nil
		})

		return err
	})

	Handle(err)

	chain := BlockChain{lastHash, db}

	return &chain
}

// create a new instance of a blockchain with a genesis block and transaction
func CreateBlockChain(address string) *BlockChain {
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		coinbaseTransaction := CoinbaseTx(address, genesisData)
		genesisBlock := genesis(coinbaseTransaction)
		fmt.Println("Genesis block created")

		err = txn.Set(genesisBlock.Hash, genesisBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesisBlock.Hash)

		lastHash = genesisBlock.Hash

		return err
	})

	Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

// create and append a new bock to the list of existing blocks
func (chain *BlockChain) AddBlock(transactions []*Transaction) {
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

	newBlock := createBlock(transactions, lastHash)

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

// locate the unspent transaction
func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTXs []Transaction

	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				// if there's a map entry
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.canBeUnlocked(address) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.isCoinbase() {
				for _, in := range tx.Inputs {
					if in.canUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Output)
					}
				}
			}
		}

		// genesis block reached
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTXs
}

// locate the unspent transaction outputs (UTXOs)
func (chain *BlockChain) FindUTXO(address string) []TransactionOutput {
	var UTXOs []TransactionOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.canBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amountToSend int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTransactions := chain.FindUnspentTransactions(address)
	accumulated := 0

	// ensure the sender has enough tokens to spend, i.e., avaiableTokens >= amountToSend
Work:
	for _, tx := range unspentTransactions {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.canBeUnlocked(address) && accumulated < amountToSend {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if accumulated >= amountToSend {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}
