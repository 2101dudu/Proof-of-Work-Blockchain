package blockchain

import (
	"fmt"
	"log"
	"strconv"
)

type BlockChain struct {
	// an array of pointers is used to ensure no copies exist
	Blocks []*Block
}

// create and append a new bock to the list of existing blocks
func (chain *BlockChain) AddBlock(data []byte) {
	chainSize := len(chain.Blocks)
	if chainSize <= 0 {
		log.Panic("Block chain not initialized")
	}

	prevBlock := chain.Blocks[chainSize-1]
	newBlock := createBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, newBlock)
}

// create a new instance of a blockchain with a genesis block
func CreateBlockChain() *BlockChain {
	return &BlockChain{[]*Block{genesis()}}
}

func (chain *BlockChain) PrintBlockChain() {
	for _, block := range chain.Blocks {
		fmt.Printf("--------\n")
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", string(block.Data))
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := NewProof(block)
		fmt.Printf("Proof-of-work: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Printf("--------\n")
	}
}
