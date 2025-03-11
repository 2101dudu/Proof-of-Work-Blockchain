package blockchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

// retrieve data from the block
// create a counter (nonce) that starts at 0
// create a hash of the data+counter
// check the hash to see if it meets a set of requirements

// requirements:
// first few bytes of the has must contain 0s

const Difficulty = 20

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)

	// left shift the bytes 256-Difficulty times
	// 256 is used because it represents the size of the block's hash
	// the number 00000000....000001 would now be 0000...0001000...000000
	//            ^256th bit       ^1st bit                 ^"difficulty"th bit
	target.Lsh(target, uint(256-Difficulty))

	return &ProofOfWork{b, target}
}

func (proof *ProofOfWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			proof.Block.PrevHash,
			proof.Block.HashTransactions(),
			toHex(int64(nonce)),
			toHex(int64(Difficulty)),
		},
		[]byte{},
	)

	return data
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0

	for nonce < math.MaxInt64 {
		// 1. prepare the data
		// 2. hash the data
		// 3. convert the hash into big.Int
		// 4. compare that big.Int with the target, inside the pow

		data := pow.InitData(nonce)
		hash = sha256.Sum256(data)

		fmt.Printf("\r%x", hash)

		intHash.SetBytes(hash[:])

		// hash met the target
		if intHash.Cmp(pow.Target) == -1 {
			fmt.Println()
			return nonce, hash[:]
		}

		nonce++
	}

	return 0, []byte{}
}

// check if the proof-of-work's parameters ensure the target is met
func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int

	data := pow.InitData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}
