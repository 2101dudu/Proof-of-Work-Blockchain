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
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
