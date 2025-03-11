package blockchain

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println("   getbalance -address ADDRESS —— get the balance for the given ADDRESS")
	fmt.Println("   createblockchain -address ADDRESS —— create a fresh blockchain and have the ADDRESS mine the genesis block")
	fmt.Println("   send -from FROM -to TO -amount AMONUT —— send AMOUNT tokens to TO from FROM")
	fmt.Println("   printblockchain —— prints the blocks in the blockchain")
}

func (cli *CommandLine) getBalance(address string) {
	chain := ContinueBlockChain()
	defer chain.Database.Close()

	amount := 0
	UTXOs := chain.FindUTXO(address)

	for _, UTXO := range UTXOs {
		amount += UTXO.Value
	}

	fmt.Printf("--------\n")
	fmt.Printf("Address %s has %d tokens\n", address, amount)
	fmt.Printf("--------\n")
}

func (cli *CommandLine) createBlockChain(address string) {
	chain := CreateBlockChain(address)
	chain.Database.Close()
	fmt.Println("blockchain created!")
}

func (cli *CommandLine) send(from, to string, amount int) {
	chain := ContinueBlockChain()
	defer chain.Database.Close()

	t := newTransaction(from, to, amount, chain)
	chain.AddBlock([]*Transaction{t})

	fmt.Printf("Sent %d tokens to %s\n", amount, to)
}

func (cli *CommandLine) printChain() {
	chain := ContinueBlockChain()
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("--------\n")
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := NewProof(block)
		fmt.Printf("Proof-of-work: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Printf("--------\n")

		// traversed the whole chain and reached the genesis block with hash = 0
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit() // badgerDB NEEDS to cleanly clean shutdown and collect garbage
	}
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockChainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	getBalanceAddresss := getBalanceCmd.String("address", "", "The address of the account you want to check the balance on")
	createBlockChainAddress := createBlockChainCmd.String("address", "", "The address of the account who will mine the genesis block")
	sendFrom := sendCmd.String("from", "", "The address of the account you want to send tokens from")
	sendTo := sendCmd.String("to", "", "The address of the account you want to send tokens to")
	sendAmount := sendCmd.Int("amount", 0, "The amount of tokens you want to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		Handle(err)
	case "createblockchain":
		err := createBlockChainCmd.Parse(os.Args[2:])
		Handle(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		Handle(err)
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddresss == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddresss)
	}

	if createBlockChainCmd.Parsed() {
		if *createBlockChainAddress == "" {
			createBlockChainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockChainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}
