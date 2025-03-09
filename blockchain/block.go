package blockchain

import (
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
