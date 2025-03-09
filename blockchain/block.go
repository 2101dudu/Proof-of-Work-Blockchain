package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strconv"
)

type Block struct {
	Hash     []byte // a hash of the Data + PrevHash
	Data     []byte
	PrevHash []byte // linked list functionality (chain)
	Nonce    int
}

// create a new instance of block with the given parameters
// create block now actually needs to be calculated and have a proof of work
func createBlock(data []byte, prevHash []byte) *Block {
	if data == nil {
		log.Panic("Cannot create a block with null data")
	}

	block := &Block{[]byte{}, data, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// create a genesis block exists — without it, the first "real" block would now have a previous block hash to reference
func genesis() *Block {
	return createBlock([]byte("GENSIS_BLOCK"), []byte{})
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
