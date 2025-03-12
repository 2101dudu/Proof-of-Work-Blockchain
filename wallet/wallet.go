package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func (w Wallet) address() []byte {
	publicKeyHashed := publicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, publicKeyHashed...)
	checksum := generateChecksum(versionedHash)

	fullHash := append(versionedHash, checksum...)

	address := base58Encode(fullHash)

	fmt.Printf("Public key: %x\n", w.PublicKey)
	fmt.Printf("Public hash: %x\n", publicKeyHashed)
	fmt.Printf("Address: %x\n", address)

	return address
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	// elliptic curve multiplication
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	return *privateKey, publicKey
}

func MakeWallet() *Wallet {
	privateKey, publicKey := newKeyPair()
	return &Wallet{PrivateKey: privateKey, PublicKey: publicKey}
}

func publicKeyHash(publicKey []byte) []byte {
	publicKeyHashed := sha256.Sum256(publicKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(publicKeyHashed[:])
	if err != nil {
		log.Panic(err)
	}

	publicKeyRipMD := hasher.Sum(nil)

	return publicKeyRipMD
}

func generateChecksum(payload []byte) []byte {
	payloadFirstHash := sha256.Sum256(payload)
	payloadSecondHash := sha256.Sum256(payloadFirstHash[:])

	return payloadSecondHash[:checksumLength]
}
