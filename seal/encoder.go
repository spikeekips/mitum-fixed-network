package seal

import (
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
)

type RLPDecodeSealType struct {
	Type   common.DataType
	Hash   rlp.RawValue
	Header rlp.RawValue
	Body   rlp.RawValue
}

type RLPEncodeSeal struct {
	Type   common.DataType
	Hash   hash.Hash
	Header Header
	Body   Body
}

type RLPDecodeSeal struct {
	Type   common.DataType
	Hash   hash.Hash
	Header Header
	Body   rlp.RawValue
}
