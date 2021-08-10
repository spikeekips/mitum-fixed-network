package valuehash

import (
	"crypto/rand"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/util"
)

func RandomSHA256() Hash {
	b := make([]byte, 100)
	_, _ = rand.Read(b)

	return NewSHA256(b)
}

func RandomSHA512() Hash {
	b := make([]byte, 100)
	_, _ = rand.Read(b)

	return NewSHA512(b)
}

// RandomSHA256WithPrefix generate random hash with string-based prefix. To
// decode, it's structure is,
// - <8: length of bytes prefix><8: length of random hash><32: random hash><bytes prefix>
//
// * 52 is max valid length of prefix
func RandomSHA256WithPrefix(prefix []byte) Hash {
	lh := util.Int64ToBytes(int64(sha256Size))
	lp := util.Int64ToBytes(int64(len(prefix)))

	return NewBytes(util.ConcatBytesSlice(lh, lp, RandomSHA256().Bytes(), prefix))
}

// RandomSHA512WithPrefix generate random hash with string-based prefix. To
// decode, it's structure is,
// - <8: length of bytes prefix><8: length of random hash><64: random hash><bytes prefix>
//
// * 20 is max valid length of prefix
func RandomSHA512WithPrefix(prefix []byte) Hash {
	lh := util.Int64ToBytes(int64(sha512Size))
	lp := util.Int64ToBytes(int64(len(prefix)))

	return NewBytes(util.ConcatBytesSlice(lh, lp, RandomSHA512().Bytes(), prefix))
}

func toString(b []byte) string {
	return base58.Encode(b)
}
