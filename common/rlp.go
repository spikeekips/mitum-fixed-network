package common

import "github.com/ethereum/go-ethereum/rlp"

func RLPEncode(i interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(i)
}

func RLPDecode(b []byte, i interface{}) error {
	return rlp.DecodeBytes(b, i)
}
