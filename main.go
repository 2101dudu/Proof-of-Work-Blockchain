package main

import (
	"golang-blockchain/blockchain"
	"os"
)

func main() {
	defer os.Exit(0)
	cli := blockchain.CommandLine{}
	cli.Run()
}
