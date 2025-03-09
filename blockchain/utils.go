package blockchain

import (
	"bytes"
	"encoding/binary"
	"log"
)

// helper function
func toHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)

	Handle(err)

	return buff.Bytes()
}

// helper error function
func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
