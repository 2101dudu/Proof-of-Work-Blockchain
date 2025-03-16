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

// serialize the transaction for later hashing
func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	Handle(err)
	return transaction
}

// hash the transaction's bytes equivalent to the transaction's ID
func (tx *Transaction) hash() []byte {
	var hash [32]byte

	// create a copy of the transaction, and set its ID to an empty byte slice
	// this is done because the ID does not come into play when hashing the transaction
	txCopy := *tx
	txCopy.ID = []byte{}

	// hash the transaction's bytes
	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// create the blockchains' first transaction — the coinbase transaction
// the coinbase includes a reward that's given to the first recepient, in this case, 100 tokens
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 24)
		_, err := rand.Read(randData)
		Handle(err)
		data = fmt.Sprintf("%x", randData)
	}

	txInput := TransactionInput{[]byte{}, -1, nil, []byte(data)}
	txOutput := NewTransactionOutput(20, to) // a reward of 20 tokens for the first miner

	tx := Transaction{nil, []TransactionInput{txInput}, []TransactionOutput{*txOutput}}
	tx.ID = tx.hash()

	return &tx
}

// create a new transaction
func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []TransactionInput
	var outputs []TransactionOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	publicKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := UTXO.FindSpendableOutputs(publicKeyHash, amount)

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
	UTXO.Blockchain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Output == -1
}

// sign the transaction using the private key
func (tx *Transaction) sign(privateKey ecdsa.PrivateKey, previousTXs map[string]Transaction) {
	// coinbase transactions don't need to be signed
	if tx.isCoinbase() {
		return
	}

	// verify that every previous transactions exist
	for _, in := range tx.Inputs {
		if previousTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction does not exist")
		}
	}

	// create a copy of the transaction, and populate it with the previous transactions
	// this is done because we need to modify the transaction in order to sign it
	txCopy := tx.trimmedCopy()

	for inId, in := range txCopy.Inputs {
		previousTX := previousTXs[hex.EncodeToString(in.ID)]

		// clear signature and set public key to the previous output's public key hash
		// this recreates the state of the transaction at signing time
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PublicKey = previousTX.Outputs[in.Output].PublicKeyHash

		// calculate the hash of this state
		txCopy.ID = txCopy.hash()

		// clear the public key as it's no longer needed
		txCopy.Inputs[inId].PublicKey = nil

		// sign the hash using the private key
		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, txCopy.ID)
		Handle(err)

		// combine the signature components (r,s) into a single byte slice
		signature := append(r.Bytes(), s.Bytes()...)

		// store the signature in the actual transaction
		tx.Inputs[inId].Signature = signature
	}
}

// create a trimmed copy of the transaction by removing the signature and public key
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

// verify a transaction using the public key
func (tx *Transaction) Verify(previousTXs map[string]Transaction) bool {
	// coinbase transactions are always valid
	if tx.isCoinbase() {
		return true
	}

	// verify that every previous transactions exist
	for _, in := range tx.Inputs {
		if previousTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction does not exist")
		}
	}

	// create a copy of the transaction, and populate it with the previous transactions
	// this is done because we need to modify the transaction in order to verify it
	txCopy := tx.trimmedCopy()
	curve := elliptic.P256()

	for inId, in := range txCopy.Inputs {
		// recreate the same state as when the transaction was signed
		previousTX := previousTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PublicKey = previousTX.Outputs[in.Output].PublicKeyHash
		txCopy.ID = txCopy.hash()
		txCopy.Inputs[inId].PublicKey = nil

		// deconstruct the signature into its components
		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		// deconstruct the public key into its coordinates
		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PublicKey)
		x.SetBytes(in.PublicKey[:(keyLen / 2)])
		y.SetBytes(in.PublicKey[(keyLen / 2):])

		// verify the signature
		rawPublicKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPublicKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

// stringify the transaction
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
