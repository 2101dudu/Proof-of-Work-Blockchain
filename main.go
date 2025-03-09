package main

import (
	"flag"
	"fmt"
	"golang-blockchain/blockchain"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct {
	blockChain *blockchain.BlockChain
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println(" add -block BLOCK_DATA — adds a block to the blockchain")
	fmt.Println(" print — Prints the blocks in the blockchain")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit() // badgerDB NEEDS to cleanly clean shutdown and collect garbage
	}
}

func (cli *CommandLine) addBlock(data string) {
	cli.blockChain.AddBlock([]byte(data))
	fmt.Println("Added the block!")
}

func (cli *CommandLine) printChain() {
	iter := cli.blockChain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("--------\n")
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", string(block.Data))
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("Proof-of-work: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Printf("--------\n")

		// traversed the whole chain and reached the genesis block with hash = 0
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) run() {
	cli.validateArgs()

	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "The block's data")

	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if addBlockCmd.Parsed() {
		// empty string
		if *addBlockData == "" {
			cli.printUsage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	defer os.Exit(0)
	chain := blockchain.CreateBlockChain()
	defer chain.Database.Close()

	cli := CommandLine{chain}
	cli.run()
}
