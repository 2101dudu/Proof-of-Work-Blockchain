package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

type Transaction struct {
	ID      []byte              // hash of the transaction
	Inputs  []TransactionInput  // inputs referecing previous transactions' outputs
	Outputs []TransactionOutput // newly created outputs
}

type TransactionInput struct {
	ID        []byte // the ID of the transaction whose outputs will serve as inputs
	Output    int    // the index of the list of outputs of that transaction
	Signature string // ownership proof
}

type TransactionOutput struct {
	Value     int    // token amount
	PublicKey string // the recepient's address
}

// hash the transaction's bytes
func (tx *Transaction) setID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// create the blockchains' first transaction — the coinbase transaction
// the coinbase includes a reward that's given to the first recepient, in this case, 100 tokens
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to) // arbitraty data
	}

	txInput := TransactionInput{[]byte{}, -1, data}
	txOutput := TransactionOutput{100, to} // a reward of 100 tokens for the first miner

	tx := Transaction{nil, []TransactionInput{txInput}, []TransactionOutput{txOutput}}
	tx.setID()

	return &tx
}

// create a new transaction
func newTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TransactionInput
	var outputs []TransactionOutput

	acc, validOutputs := chain.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txId, outputs := range validOutputs {
		txID, err := hex.DecodeString(txId)
		Handle(err)

		for _, out := range outputs {
			input := TransactionInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TransactionOutput{amount, to})

	// if we have tokens leftover, we need to point them to ourselves
	if acc > amount {
		outputs = append(outputs, TransactionOutput{acc - amount, from})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.setID()

	return &tx
}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Output == -1
}

func (txIn *TransactionInput) canUnlock(address string) bool {
	return txIn.Signature == address
}

func (txOut *TransactionOutput) canBeUnlocked(publicKey string) bool {
	return txOut.PublicKey == publicKey
}
