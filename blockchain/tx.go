package blockchain

import (
	"bytes"
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

func NewTransactionOutput(value int, address string) *TransactionOutput {
	txOut := &TransactionOutput{value, nil}
	txOut.lock([]byte(address))

	return txOut
}

func (in *TransactionInput) usesKey(publicKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PublicKey)

	return bytes.Compare(publicKeyHash, lockingHash) == 0
}

func (out *TransactionOutput) lock(address []byte) {
	publicKeyHash := wallet.Base58Decode(address)

	//remove version and checksum
	publicKeyHash = publicKeyHash[1 : len(publicKeyHash)-4]
	out.PublicKeyHash = publicKeyHash
}

func (out *TransactionOutput) isLockedWithKey(publicKeyHash []byte) bool {
	return bytes.Compare(out.PublicKeyHash, publicKeyHash) == 0
}
