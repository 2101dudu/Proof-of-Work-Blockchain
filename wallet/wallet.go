package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"

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

// implement custom JSON marshalling for the wallel
func (w Wallet) MarshalJSON() ([]byte, error) {
	// create a helper struct with hex representations of the key components.
	type walletJSON struct {
		D            string `json:"d"`
		PublicKeyX   string `json:"publicKeyX"`
		PublicKeyY   string `json:"publicKeyY"`
		RawPublicKey string `json:"rawPublicKey"`
	}
	temp := walletJSON{
		D:            w.PrivateKey.D.Text(16),
		PublicKeyX:   w.PrivateKey.PublicKey.X.Text(16),
		PublicKeyY:   w.PrivateKey.PublicKey.Y.Text(16),
		RawPublicKey: hex.EncodeToString(w.PublicKey),
	}
	return json.Marshal(temp)
}

// implement custom JSON unmarshalling for the wallet
func (w *Wallet) UnmarshalJSON(data []byte) error {
	// define a helper struct matching the one in MarshalJSON
	type walletJSON struct {
		D            string `json:"d"`
		PublicKeyX   string `json:"publicKeyX"`
		PublicKeyY   string `json:"publicKeyY"`
		RawPublicKey string `json:"rawPublicKey"`
	}
	var temp walletJSON
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	d := new(big.Int)
	d.SetString(temp.D, 16)
	x := new(big.Int)
	x.SetString(temp.PublicKeyX, 16)
	y := new(big.Int)
	y.SetString(temp.PublicKeyY, 16)

	// reconstruct the ecdsa.PrivateKey using P256 as the curve
	w.PrivateKey = ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		},
		D: d,
	}

	raw, err := hex.DecodeString(temp.RawPublicKey)
	if err != nil {
		return err
	}
	w.PublicKey = raw

	return nil
}

func (w Wallet) address() []byte {
	publicKeyHashed := PublicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, publicKeyHashed...)
	checksum := generateChecksum(versionedHash)

	fullHash := append(versionedHash, checksum...)

	address := Base58Encode(fullHash)

	return address
}

// Address: 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
// convert into a full hash
// Full hash: 00248bd9e7a51b7dd07aba9766a7c62d50207902802bc6c767
// extract the components
// [Version] 00
// [PublicKeyHash] 248bd9e7a51b7dd07aba9766a7c62d5020790280
// [Checksum] 2bc6c767
//
// ValidateAddress checks if a given address is valid by:
// 1. Decoding the Base58 address to get the full hash
// 2. Extracting the version byte and actual checksum
// 3. Generating a checksum from the version and public key hash
// 4. Comparing the actual and generated checksums
func ValidateAddress(address string) bool {
	// decode the Base58 address back into the full hash
	publicKeyHash := Base58Decode([]byte(address))

	// extract the actual checksum (last 4 bytes)
	actualChecksum := publicKeyHash[len(publicKeyHash)-checksumLength:]

	// extract the version byte (first byte)
	version := publicKeyHash[0]

	// extract the public key hash (between the version and checksum)
	publicKeyHash = publicKeyHash[1 : len(publicKeyHash)-checksumLength]

	// generate checksum from version + public key hash
	targetChecksum := generateChecksum(append([]byte{version}, publicKeyHash...))

	// compare the actual and generated checksums
	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	// concatenate X and Y coordinates to form the public key
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	return *privateKey, publicKey
}

func MakeWallet() *Wallet {
	privateKey, publicKey := newKeyPair()
	return &Wallet{PrivateKey: privateKey, PublicKey: publicKey}
}

func PublicKeyHash(publicKey []byte) []byte {
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
