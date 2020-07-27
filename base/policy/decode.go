package policy

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodePolicyV0(enc encoder.Encoder, b []byte) (PolicyV0, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return PolicyV0{}, err
	} else if i == nil {
		return PolicyV0{}, nil
	} else if v, ok := i.(PolicyV0); !ok {
		return PolicyV0{}, hint.InvalidTypeError.Errorf("not PolicyV0; type=%T", i)
	} else {
		return v, nil
	}
}
