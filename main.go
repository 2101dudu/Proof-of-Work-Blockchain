package main

import (
	"bytes"
	"crypto/sha256"
	"log"
)

type BlockChain struct {
	// an array of pointers is used to ensure no copys exist
	blocks []*Block
}

type Block struct {
	Hash     []byte // hash(data + PrevHash)
	Data     []byte
	PrevHash []byte // linked list functionality (chain)
}

func (b *Block) deriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte("gello"))
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

// TODO: ensure a genesis block exists
func createBlock(data []byte, prevHash []byte) *Block {
	if data == nil {
		log.Panic("Cannot create a block with null data")
	}

	block := &Block{[]byte{}, data, prevHash}
	block.deriveHash()
	return block
}

func (chain *BlockChain) addBlock(data []byte) {
	prevBlock := chain.blocks[len(chain.blocks)-1]
	newBlock := createBlock(data, prevBlock.Hash)
	chain.blocks = append(chain.blocks, newBlock)
}

func main() {

}
