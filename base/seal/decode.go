package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeSeal(enc encoder.Encoder, b []byte) (Seal, error) {
	if hinter, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if s, ok := hinter.(Seal); !ok {
		return nil, util.WrongTypeError.Errorf("not seal.Seal; type=%T", hinter)
	} else {
		return s, nil
	}
}
