package key

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodePublickey(enc encoder.Encoder, b []byte) (Publickey, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Publickey); !ok {
		return nil, hint.InvalidTypeError.Errorf("not key.Publickey; type=%T", i)
	} else {
		return v, nil
	}
}
