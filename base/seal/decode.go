package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeSeal(b []byte, enc encoder.Encoder) (Seal, error) {
	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if s, ok := hinter.(Seal); !ok {
		return nil, util.WrongTypeError.Errorf("not seal.Seal; type=%T", hinter)
	} else {
		return s, nil
	}
}
