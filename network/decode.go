package network

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodeNodeInfo(enc encoder.Encoder, b []byte) (NodeInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(NodeInfo); !ok {
		return nil, hint.InvalidTypeError.Errorf("not NodeInfo; type=%T", i)
	} else {
		return v, nil
	}
}
