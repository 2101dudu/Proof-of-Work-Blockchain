package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"golang-blockchain/wallet"
	"log"
	"math/big"
	"strings"
)

type Transaction struct {
	ID      []byte              // hash of the transaction
	Inputs  []TransactionInput  // inputs referecing previous transactions' outputs
	Outputs []TransactionOutput // newly created outputs
}

func (tx *Transaction) serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.serialize())

	return hash[:]
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

	txInput := TransactionInput{[]byte{}, -1, nil, []byte(data)}
	txOutput := NewTransactionOutput(100, to) // a reward of 100 tokens for the first miner

	tx := Transaction{nil, []TransactionInput{txInput}, []TransactionOutput{*txOutput}}
	tx.setID()

	return &tx
}

// create a new transaction
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TransactionInput
	var outputs []TransactionOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	publicKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := chain.FindSpendableOutputs(publicKeyHash, amount)

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txId, outputs := range validOutputs {
		txID, err := hex.DecodeString(txId)
		Handle(err)

		for _, out := range outputs {
			input := TransactionInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTransactionOutput(amount, to))

	// if we have tokens leftover, we need to point them to ourselves
	if acc > amount {
		outputs = append(outputs, *NewTransactionOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.hash()
	chain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Output == -1
}

func (tx *Transaction) sign(privateKey ecdsa.PrivateKey, previousTXs map[string]Transaction) {
	if tx.isCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if previousTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction does not exist")
		}
	}

	txCopy := tx.trimmedCopy()

	for inId, in := range txCopy.Inputs {
		previousTX := previousTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PublicKey = previousTX.Outputs[in.Output].PublicKeyHash
		txCopy.ID = txCopy.hash()
		txCopy.Inputs[inId].PublicKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, txCopy.ID)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Inputs[inId].Signature = signature
	}
}

func (tx *Transaction) trimmedCopy() Transaction {
	var inputs []TransactionInput
	var outputs []TransactionOutput

	for _, in := range tx.Inputs {
		// trimming out the signature and the public key
		inputs = append(inputs, TransactionInput{in.ID, in.Output, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TransactionOutput{out.Value, out.PublicKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Verify(previousTXs map[string]Transaction) bool {
	if tx.isCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if previousTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction does not exist")
		}
	}

	txCopy := tx.trimmedCopy()
	curve := elliptic.P256()

	for inId, in := range txCopy.Inputs {
		previousTX := previousTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PublicKey = previousTX.Outputs[in.Output].PublicKeyHash
		txCopy.ID = txCopy.hash()
		txCopy.Inputs[inId].PublicKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PublicKey)
		x.SetBytes(in.Signature[:(keyLen / 2)])
		y.SetBytes(in.Signature[(keyLen / 2):])

		rawPublicKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPublicKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Output))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PublicKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PublicKeyHash))
	}

	return strings.Join(lines, "\n")
}
