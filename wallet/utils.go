package wallet

import (
	"log"

	"github.com/mr-tron/base58"
)

// base 58 is similar to base 64, only that its missing the following characters:
// 0 O l I + /
// this was done because addresses are the main way that people send tokens to,
// addresses with those characters can make the addresses wrongly interpreted and misstyped

func base58Encode(input []byte) []byte {
	return []byte(base58.Encode(input))
}

func base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input))
	if err != nil {
		log.Panic(err)
	}

	return decode
}
