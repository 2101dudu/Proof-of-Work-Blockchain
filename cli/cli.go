package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"golang-blockchain/blockchain"
	"golang-blockchain/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println("   getbalance -address ADDRESS —— get the balance for the given ADDRESS")
	fmt.Println("   createblockchain -address ADDRESS —— create a fresh blockchain and have the ADDRESS mine the genesis block")
	fmt.Println("   send -from FROM -to TO -amount AMONUT —— send AMOUNT tokens to TO from FROM")
	fmt.Println("   printchain —— prints the blocks in the blockchain")
	fmt.Println("   createwallet —— create a new wallet")
	fmt.Println("   listaddresses —— list the addresses in the wallet file")
	fmt.Println("   reindexutxo —— rebuild the UTXO set")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.ContinueBlockChain()
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	amount := 0
	publicKeyHash := wallet.Base58Decode([]byte(address))
	publicKeyHash = publicKeyHash[1 : len(publicKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(publicKeyHash)

	for _, UTXO := range UTXOs {
		amount += UTXO.Value
	}

	fmt.Printf("--------\n")
	fmt.Printf("Address %s has %d tokens\n", address, amount)
	fmt.Printf("--------\n")
}

func (cli *CommandLine) createBlockChain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.CreateBlockChain(address)
	chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	fmt.Println("blockchain created!")
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is invalid")
	}

	if !wallet.ValidateAddress(to) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.ContinueBlockChain()
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	t := blockchain.NewTransaction(from, to, amount, &UTXOSet)
	block := chain.AddBlock([]*blockchain.Transaction{t})
	UTXOSet.Update(block)

	fmt.Printf("Sent %d tokens to %s\n", amount, to)
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("--------\n")
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("Proof-of-work: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("--------\n")

		// traversed the whole chain and reached the genesis block with hash = 0
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createWallet() {
	wallets, _ := wallet.CreateWallets()
	address := wallets.AddWallet()
	wallets.SaveFile()

	fmt.Printf("The address of your new wallet: %s\n", address)
}

func (cli *CommandLine) listAddresses() {
	wallets, _ := wallet.CreateWallets()
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) reindexUTXO() {
	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are now %d transactions in the UTXO set.\n", count)
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
	createwalletcmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listaddressescmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reeindexUTXOcmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)

	getBalanceAddresss := getBalanceCmd.String("address", "", "The address of the account you want to check the balance on")
	createBlockChainAddress := createBlockChainCmd.String("address", "", "The address of the account who will mine the genesis block")
	sendFrom := sendCmd.String("from", "", "The address of the account you want to send tokens from")
	sendTo := sendCmd.String("to", "", "The address of the account you want to send tokens to")
	sendAmount := sendCmd.Int("amount", 0, "The amount of tokens you want to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "createblockchain":
		err := createBlockChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "createwallet":
		err := createwalletcmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "listaddresses":
		err := listaddressescmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "reindexutxo":
		err := reeindexUTXOcmd.Parse(os.Args[2:])
		blockchain.Handle(err)
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

	if createwalletcmd.Parsed() {
		cli.createWallet()
	}

	if listaddressescmd.Parsed() {
		cli.listAddresses()
	}

	if reeindexUTXOcmd.Parsed() {
		cli.reindexUTXO()
	}
}
