package blockchain

import (
	"bytes"
	"encoding/gob"
	"golang-blockchain/wallet"
)

type TransactionInput struct {
	ID        []byte // the ID of the transaction whose outputs will serve as inputs
	Output    int    // the index of the list of outputs of that transaction
	Signature []byte // the hashed key of the owner of the referenced output
	PublicKey []byte // the public key of the owner of the referenced output
}

type TransactionOutput struct {
	Value         int    // token amount
	PublicKeyHash []byte // the hashed recepient's address
}

type TransactionOutputs struct {
	Outputs []TransactionOutput
}

func NewTransactionOutput(value int, address string) *TransactionOutput {
	txOut := &TransactionOutput{value, nil}

	// lock the output to the address by parameterizing the public key hash
	txOut.lock([]byte(address))

	return txOut
}

func (out *TransactionOutput) lock(address []byte) {
	publicKeyHash := wallet.Base58Decode(address)

	// remove version and checksum
	publicKeyHash = publicKeyHash[1 : len(publicKeyHash)-4]
	out.PublicKeyHash = publicKeyHash
}

// check if the output is locked with the given public key hash
func (out *TransactionOutput) isLockedWithKey(publicKeyHash []byte) bool {
	return bytes.Compare(out.PublicKeyHash, publicKeyHash) == 0
}

func (outs TransactionOutputs) Serialize() []byte {
	var buffer bytes.Buffer

	encode := gob.NewEncoder(&buffer)
	err := encode.Encode(outs)
	Handle(err)

	return buffer.Bytes()
}

func DeserializeOutputs(data []byte) TransactionOutputs {
	var outputs TransactionOutputs

	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	Handle(err)

	return outputs
}
