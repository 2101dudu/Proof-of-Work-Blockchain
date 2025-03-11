package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type Block struct {
	Hash         []byte // a hash of the Data + PrevHash
	Transactions []*Transaction
	PrevHash     []byte // linked list functionality (chain)
	Nonce        int
}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

// create a new instance of block with the given parameters
// create block now actually needs to be calculated and have a proof of work
func createBlock(transactions []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, transactions, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// create a genesis block exists — without it, the first "real" block would now have a previous block hash to reference
func genesis(coinbase *Transaction) *Block {
	return createBlock([]*Transaction{coinbase}, []byte{})
}

// GO's BadgerDB requires byte slices, so a Serialize() needs to exist
func (b *Block) Serialize() []byte {
	var buf bytes.Buffer
	enconder := gob.NewEncoder(&buf)
	err := enconder.Encode(b)

	Handle(err)

	return buf.Bytes()
}

// GO's BadgerDB requires byte slices, so a Deserialize() needs to exist
func Deserialize(data []byte) (b *Block) {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)

	Handle(err)

	return &block
}
