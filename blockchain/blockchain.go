package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

// helper function to check if MANIFEST file exists, i.e., the DB
func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}

// fetch existing blockchain and continue the chain
func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if !DBexists(path) {
		fmt.Println("BlockChain has not been created yet. Create one!")
		runtime.Goexit()
	}

	var lastHash []byte

	opts := badger.DefaultOptions(path)
	opts.ValueDir = path
	db, err := openDB(path, opts)
	Handle(err)

	// fetch blockchains' last hash pointer
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(v)
			return nil
		})

		return err
	})

	Handle(err)

	chain := BlockChain{lastHash, db}

	return &chain
}

// create a new instance of a blockchain with a genesis block and transaction
func CreateBlockChain(address, nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if DBexists(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	var lastHash []byte

	opts := badger.DefaultOptions(path)
	opts.ValueDir = path
	db, err := openDB(path, opts)
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

func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := chain.Iterator()

	for {
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (chain *BlockChain) GetBestHeight() int {
	var lastBlock Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		var lastHash []byte
		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(v)
			return nil
		})

		item, err = txn.Get(lastHash)

		var lastBlockData []byte
		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastBlockData = slices.Clone(v)
			return nil
		})

		lastBlock = *Deserialize(lastBlockData)

		return err
	})
	Handle(err)

	return lastBlock.Height
}

// create and append a new bock to the list of existing blocks
func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	// fetch blockchains' last hash pointer
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(v)
			return nil
		})

		item, err = txn.Get(lastHash)

		var lastBlockData []byte
		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastBlockData = slices.Clone(v)
			return nil
		})

		lastBlock := Deserialize(lastBlockData)

		lastHeight = lastBlock.Height

		return err
	})

	newBlock := createBlock(transactions, lastHash, lastHeight+1)
	fmt.Println("lastheight is", lastHeight+1)

	// set blockchains' last hash pointer
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handle(err)

	return newBlock
}

func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err != nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		Handle(err)

		item, err := txn.Get([]byte("lh"))
		Handle(err)

		var lastHash []byte
		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastHash = slices.Clone(v)
			return nil
		})

		item, err = txn.Get(lastHash)

		var lastBlockData []byte
		err = item.Value(func(v []byte) error {
			// this func with val would only be called if item.Value() encounters no error.
			lastBlockData = slices.Clone(v)
			return nil
		})

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			Handle(err)
			chain.LastHash = block.Hash
		}

		return nil
	})
	Handle(err)
}

func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Could not find block")
		} else {
			var blockData []byte
			err = item.Value(func(v []byte) error {
				// this func with val would only be called if item.Value() encounters no error.
				blockData = slices.Clone(v)
				return nil
			})

			block = *Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

// locate the unspent transaction
func (chain *BlockChain) FindUnspentTransactions() map[string]TransactionOutputs {
	UTXO := make(map[string]TransactionOutputs)

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
				// if the output that has been spent, skip to the next iteration
				// and dont add it to the unspent transactions slice
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if !tx.isCoinbase() {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Output)
				}
			}
		}
		// genesis block reached
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXO
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
	if tx.isCoinbase() {
		return true
	}

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

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)

	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}
