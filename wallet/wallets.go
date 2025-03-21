package wallet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const walletFile = "./tmp/wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet `json:"wallets"`
}

func CreateWallets(nodeId string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.loadFile(nodeId)
	return &wallets, err
}

func (wallets *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	wallets.Wallets[address] = wallet

	return address
}

func (wallets *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range wallets.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (wallets *Wallets) GetWallet(address string) Wallet {
	return *wallets.Wallets[address]
}

func (wallets *Wallets) loadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var ws Wallets
	err = json.Unmarshal(fileContent, &ws)
	if err != nil {
		return err
	}

	wallets.Wallets = ws.Wallets
	return nil
}

func (wallets *Wallets) SaveFile(nodeId string) {
	walletFile := fmt.Sprintf(walletFile, nodeId)

	data, err := json.MarshalIndent(wallets, "", "  ")
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, data, 0644)
	if err != nil {
		log.Panic(err)
	}
}
