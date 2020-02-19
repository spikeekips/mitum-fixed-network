package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

func decodeHash(enc encoder.Encoder, b []byte) (valuehash.Hash, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(valuehash.Hash); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		return v, nil
	}
}

func decodePublickey(enc encoder.Encoder, b []byte) (key.Publickey, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(key.Publickey); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not key.Publickey; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeAddress(enc encoder.Encoder, b []byte) (Address, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Address); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not Address; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeFact(enc encoder.Encoder, b []byte) (Fact, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Fact); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not Fact; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeVoteproof(enc encoder.Encoder, b []byte) (Voteproof, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Voteproof); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not Voteproof; type=%T", i)
	} else {
		return v, nil
	}
}
