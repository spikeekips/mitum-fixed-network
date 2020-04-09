package valuehash

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func Decode(enc encoder.Encoder, b []byte) (Hash, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Hash); !ok {
		return nil, hint.InvalidTypeError.Errorf("not valuehash.Hash; type=%T", i)
	} else {
		return v, nil
	}
}
