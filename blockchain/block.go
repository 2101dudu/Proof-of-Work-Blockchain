package blockchain

import (
	"bytes"
	"encoding/gob"
)

type Block struct {
	Hash         []byte         // a hash of the Data + PrevHash
	Transactions []*Transaction // the data of a block
	PrevHash     []byte         // linked list functionality (chain)
	Nonce        int
}

// helper function to hash the blocks' transactions
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.serialize())
	}
	tree := newMerkleTree(txHashes)

	return tree.RootNode.Data
}

// create a new instance of block with the given parameters
func createBlock(transactions []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, transactions, prevHash, 0}
	pow := NewProof(block) // proove block's creation
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
