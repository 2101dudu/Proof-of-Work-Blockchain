package main

import (
	"golang-blockchain/blockchain"
	"os"
)

func main() {
	defer os.Exit(0)
	chain := blockchain.CreateBlockChain()
	defer chain.Database.Close()

	cli := blockchain.CommandLine{BlockChain: chain}
	cli.Run()
}
