package iostdb

import (
	"crypto/sha256"
	"github.com/btcsuite/btcutil/base58"
)

func Sha256(raw []byte) []byte {
	var data = sha256.Sum256(raw)
	return data[:]
}

func Hash160(raw []byte) []byte {
	var data = sha256.Sum256(raw)
	return ripemd160.New().Sum(data[len(data):])
}

func Base58Encode(raw []byte) string {
	return base58.Encode(raw)
}






