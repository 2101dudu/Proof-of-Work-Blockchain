package main

import block "golang-blockchain/blockchain"

func main() {
	blockChain := block.CreateBlockChain()

	blockChain.AddBlock([]byte("Hello blockchain"))
	blockChain.AddBlock([]byte("I'm the third block"))
	blockChain.AddBlock([]byte("and I'm the fourth block"))

	blockChain.PrintBlockChain()
}
