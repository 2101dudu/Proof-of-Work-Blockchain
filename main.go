package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log"
)

type BlockChain struct {
	// an array of pointers is used to ensure no copies exist
	blocks []*Block
}

type Block struct {
	Hash     []byte // a hash of the Data + PrevHash
	Data     []byte
	PrevHash []byte // linked list functionality (chain)
}

// compute the hash from the Data + PrevHash values
func (b *Block) deriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte("gello"))
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

// create a new instance of block with the given parameters, fill it in with its hash
func createBlock(data []byte, prevHash []byte) *Block {
	if data == nil {
		log.Panic("Cannot create a block with null data")
	}

	block := &Block{[]byte{}, data, prevHash}
	block.deriveHash()
	return block
}

// create and append a new bock to the list of existing blocks
func (chain *BlockChain) addBlock(data []byte) {
	chainSize := len(chain.blocks)
	if chainSize <= 0 {
		log.Panic("Block chain not initialized")
	}

	prevBlock := chain.blocks[chainSize-1]
	newBlock := createBlock(data, prevBlock.Hash)
	chain.blocks = append(chain.blocks, newBlock)
}

// create a genesis block exists — without it, the first "real" block would now have a previous block hash to reference
func genesis() *Block {
	return createBlock([]byte("GENSIS_BLOCK"), []byte{})
}

// TODO: revisit blockchain creation logic — an append to an empty list may not be necessary

// create a new instance of a blockchain with a genesis block
func createBlockChain() *BlockChain {
	return &BlockChain{append([]*Block{}, genesis())}
}

func (chain *BlockChain) printBlockChain() {
	for _, block := range chain.blocks {
		fmt.Printf("--------\n")
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", string(block.Data))
		fmt.Printf("Current Hash: %x\n", block.Hash)
		fmt.Printf("--------\n")
	}
}

func main() {
	blockChain := createBlockChain()

	blockChain.addBlock([]byte("Hello blockchain"))
	blockChain.addBlock([]byte("I'm the third block"))
	blockChain.addBlock([]byte("and I'm the forth block"))

	blockChain.printBlockChain()
}
