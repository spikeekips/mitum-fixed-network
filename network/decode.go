package network

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeNodeInfo(b []byte, enc encoder.Encoder) (NodeInfo, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(NodeInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not NodeInfo; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeConnInfo(b []byte, enc encoder.Encoder) (ConnInfo, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(ConnInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not ConnInfo; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeHandoverSeal(b []byte, enc encoder.Encoder) (HandoverSeal, error) {
	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if s, ok := hinter.(HandoverSeal); !ok {
		return nil, util.WrongTypeError.Errorf("not Handover; type=%T", hinter)
	} else {
		return s, nil
	}
}
