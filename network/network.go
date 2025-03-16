package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"golang-blockchain/blockchain"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/vrecan/death/v3"
)

const (
	protocol      = "tcp"
	version       = 1
	commandLength = 12
)

var (
	nodeAddress     string
	minerAddress    string
	knownAddress    string
	KnownNodes      = []string{"localhost:3001"}
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
)

type Address struct {
	AddressList []string
}

type Block struct {
	AddressFrom string
	Block       []byte
}

type GetBlocks struct {
	AddressFrom string
}

type GetData struct {
	AddressFrom string
	Type        string
	ID          []byte
}

type Inventory struct {
	AddressFrom string
	Type        string
	Items       [][]byte
}

type Transaction struct {
	AddressFrom string
	Transaction []byte
}

type Version struct {
	AddressFrom string
	Version     int
	BestHeight  int
}

// 1. Create blockchain
// 2. Wallet connects and downloads blockchain
// 3. Miner connects and downloads blockchain
// 4. Wallet creates transaction
// 5. Minet gets transaction to memory pool
// 5.1. Enough transactions -> mine block
// 6. Block gets sent to the central node
// 7. Wallet syncs and verifies the node's payments were successfull

func CmdToBytes(cmd string) []byte {
	var bytes [commandLength]byte

	for i, c := range cmd {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}

	return fmt.Sprintf("%s", cmd)
}

func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlocks(node)
	}
}

func ExtractCmd(request []byte) []byte {
	return request[:commandLength]
}

func SendAddr(addr string) {
	nodes := Address{AddressList: KnownNodes}
	nodes.AddressList = append(nodes.AddressList, nodeAddress)
	payload := GobEncode(nodes)
	request := append(CmdToBytes("addr"), payload...)

	SendData(addr, request)
}

func SendBlock(addr string, block *blockchain.Block) {
	data := Block{AddressFrom: addr, Block: block.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("block"), payload...)

	SendData(addr, request)
}

func SendInventory(addr, kind string, items [][]byte) {
	data := Inventory{AddressFrom: addr, Type: kind, Items: items}
	payload := GobEncode(data)
	request := append(CmdToBytes("inv"), payload...)

	SendData(addr, request)
}

func SendTransaction(addr string, tx *blockchain.Transaction) {
	data := Transaction{AddressFrom: addr, Transaction: tx.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("inv"), payload...)

	SendData(addr, request)
}

func SendVersion(addr string, chain *blockchain.BlockChain) {
	// TODO:
	bestHeight := chain.GetBestHeight()
	data := Version{AddressFrom: addr, Version: version, BestHeight: bestHeight}
	payload := GobEncode(data)
	request := append(CmdToBytes("version"), payload...)

	SendData(addr, request)
}

func SendGetBlocks(addr string) {
	data := GetBlocks{AddressFrom: addr}
	payload := GobEncode(data)
	request := append(CmdToBytes("getBlocks"), payload...)

	SendData(addr, request)
}

func SendGetData(addr, kind string, id []byte) {
	data := GetData{AddressFrom: addr, Type: kind, ID: id}
	payload := GobEncode(data)
	request := append(CmdToBytes("getData"), payload...)

	SendData(addr, request)
}

func SendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Println("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func HandleAddress(request []byte) {
	var buff bytes.Buffer
	var payload Address

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	KnownNodes = append(KnownNodes, payload.AddressList...)
	fmt.Println("There are %d known nodes\n", len(KnownNodes))
	RequestBlocks()
}

func HandleBlock(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := blockchain.Deserialize(blockData)

	fmt.Println("Received a new block!\n")
	chain.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddressFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := blockchain.UTXOSet{chain}
		UTXOSet.Reindex()
	}
}

func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := chain.GetBlockHashes()
	SendInventory(payload.AddressFrom, "block", blocks)
}

func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddressFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendTransaction(payload.AddressFrom, &tx)
	}
}
func HandleTx(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Transaction

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d\n", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddressFrom {
				SendInventory(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(minerAddress) > 0 {
			MineTx(chain)
		}
	}
}

func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All Transactions are invalid")
		return
	}

	cbTx := blockchain.CoinbaseTx(minerAddress, "")
	txs = append(txs, cbTx)

	newBlock := chain.MineBlock(txs)
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	UTXOSet.Reindex()

	fmt.Println("New Block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			SendInventory(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}

func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight {
		SendGetBlocks(payload.AddressFrom)
	} else if bestHeight > otherHeight {
		SendVersion(payload.AddressFrom, chain)
	}

	if !NodeIsKnown(payload.AddressFrom) {
		KnownNodes = append(KnownNodes, payload.AddressFrom)
	}
}

func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inventory

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddressFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddressFrom, "tx", txID)
		}
	}
}

func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := io.ReadAll(conn)
	defer conn.Close()

	if err != nil {
		log.Panic(err)
	}

	command := BytesToCmd(req[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		HandleAddress(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInv(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)
	case "tx":
		HandleTx(req, chain)
	case "version":
		HandleVersion(req, chain)
	default:
		fmt.Println("Unknown command")
	}
}

func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprint("localhost:%s", nodeID)
	minerAddress = minerAddress

	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	chain := blockchain.ContinueBlockChain()
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleConnection(conn, chain)
	}
}

func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func CloseDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}
