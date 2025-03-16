package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"golang-blockchain/blockchain"
	"golang-blockchain/network"
	"golang-blockchain/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println("   getbalance -address ADDRESS —— get the balance for the given ADDRESS")
	fmt.Println("   createblockchain -address ADDRESS —— create a fresh blockchain and have the ADDRESS mine the genesis block")
	fmt.Println("   send -from FROM -to TO -amount AMOUNT -mine —— Send amount of coins. If -mine flag is set, mine off of this node")
	fmt.Println("   printchain —— prints the blocks in the blockchain")
	fmt.Println("   createwallet —— create a new wallet")
	fmt.Println("   listaddresses —— list the addresses in the wallet file")
	fmt.Println("   reindexutxo —— rebuild the UTXO set")
	fmt.Println("   startnode -miner ADDRESS —— Start a node with ID specified in NODE_ID .env variable; miner enables mining")
}

func (cli *CommandLine) getBalance(address string, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.ContinueBlockChain(nodeID)
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

func (cli *CommandLine) createBlockChain(address, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.CreateBlockChain(address, nodeID)
	chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	fmt.Println("blockchain created!")
}

func (cli *CommandLine) send(from, to string, amount int, nodeID string, mineNow bool) {
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is invalid")
	}

	if !wallet.ValidateAddress(to) {
		log.Panic("Address is invalid")
	}

	chain := blockchain.ContinueBlockChain(nodeID)
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx := blockchain.NewTransaction(&wallet, to, amount, &UTXOSet)
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	} else {
		network.SendTransaction(network.KnownNodes[0], tx)
		fmt.Println("Sent transaction")
	}

	fmt.Printf("Sent %d tokens to %s\n", amount, to)
}

func (cli *CommandLine) printChain(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
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

func (cli *CommandLine) createWallet(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeID)

	fmt.Printf("The address of your new wallet: %s\n", address)
}

func (cli *CommandLine) listAddresses(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) reindexUTXO(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are now %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	network.StartServer(nodeID, minerAddress)
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit() // badgerDB NEEDS to cleanly clean shutdown and collect garbage
	}
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockChainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createwalletcmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listaddressescmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reeindexUTXOcmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddresss := getBalanceCmd.String("address", "", "The address of the account you want to check the balance on")
	createBlockChainAddress := createBlockChainCmd.String("address", "", "The address of the account who will mine the genesis block")
	sendFrom := sendCmd.String("from", "", "The address of the account you want to send tokens from")
	sendTo := sendCmd.String("to", "", "The address of the account you want to send tokens to")
	sendAmount := sendCmd.Int("amount", 0, "The amount of tokens you want to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

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
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
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
		cli.getBalance(*getBalanceAddresss, nodeID)
	}

	if createBlockChainCmd.Parsed() {
		if *createBlockChainAddress == "" {
			createBlockChainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockChainAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createwalletcmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listaddressescmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if reeindexUTXOcmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}
}
