package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
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

// helper function to check if MANIFEST file exists, i.e., the DB
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// fetch existing blockchain and continue the chain
func ContinueBlockChain() *BlockChain {
	var lastHash []byte

	if !DBexists() {
		fmt.Println("BlockChain has not been created yet. Create one!")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handle(err)

	// fetch blockchains' last hash pointer
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

	// set blockchains' last hash pointer
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

	// fetch blockchains' last hash pointer
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

	// set blockchains' last hash pointer
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
func (chain *BlockChain) FindUnspentTransactions(publicKeyHash []byte) []Transaction {
	var unspentTXs []Transaction

	// create a map to track the transaction's IDs (string) whose outputs' indices (int) have been spent
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	// iterate over the blockchain
	for {
		block := iter.Next()

		// iterate over the block's transactions
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				// if the output that regards the address has been spent, skip to the next iteration
				// and dont add it to the unspent transactions slice
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				// if the code reaches this line, the blocks' output has not been spend
				// if the output regards the address in question, add the transaction to the slice
				if out.isLockedWithKey(publicKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.isCoinbase() {
				// add inputs regarding the address to the slice of spend tokens
				for _, in := range tx.Inputs {
					if in.usesKey(publicKeyHash) {
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
func (chain *BlockChain) FindUTXO(publicKeyHash []byte) []TransactionOutput {
	var UTXOs []TransactionOutput
	unspentTransactions := chain.FindUnspentTransactions(publicKeyHash)

	// only retrieve the unspent outputs
	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.isLockedWithKey(publicKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// retrieve amount of tokens aswell as the transactions' IDs whose outputs concern the address' recipient
func (chain *BlockChain) FindSpendableOutputs(publicKeyHash []byte, amountToSend int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTransactions := chain.FindUnspentTransactions(publicKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTransactions {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.isLockedWithKey(publicKeyHash) && accumulated < amountToSend {
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

// locate a transaction by its ID
func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist")
}

// sign a transaction using the private key
func (chain *BlockChain) SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey) {
	previousTXs := make(map[string]Transaction)

	// locate every previous transaction that is referenced by the input
	for _, in := range tx.Inputs {
		previousTX, err := chain.FindTransaction(in.ID)
		Handle(err)
		previousTXs[hex.EncodeToString(previousTX.ID)] = previousTX
	}

	// sign the previous transactions using the private key
	tx.sign(privateKey, previousTXs)
}

// verify a transaction using the public key
func (chain *BlockChain) VerifyTransaction(tx *Transaction) bool {
	previousTXs := make(map[string]Transaction)

	// locate every previous transaction that is referenced by the input
	for _, in := range tx.Inputs {
		previousTX, err := chain.FindTransaction(in.ID)
		Handle(err)
		previousTXs[hex.EncodeToString(previousTX.ID)] = previousTX
	}

	// verify the transaction
	return tx.Verify(previousTXs)
}
