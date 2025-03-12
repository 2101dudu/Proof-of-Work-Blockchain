package wallet

import (
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
	publicKeyHashed := publicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, publicKeyHashed...)
	checksum := generateChecksum(versionedHash)

	fullHash := append(versionedHash, checksum...)

	address := base58Encode(fullHash)

	return address
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
