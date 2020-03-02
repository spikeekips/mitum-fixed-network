package valuehash

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
)

func Decode(enc encoder.Encoder, b []byte) (Hash, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Hash); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		return v, nil
	}
}
