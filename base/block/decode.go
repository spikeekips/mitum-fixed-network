package block

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeManifest(enc encoder.Encoder, b []byte) (Manifest, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Manifest); !ok {
		return nil, util.WrongTypeError.Errorf("not Manifest; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeConsensusInfo(enc encoder.Encoder, b []byte) (ConsensusInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(ConsensusInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not ConsensusInfoifest; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeSuffrageInfo(enc encoder.Encoder, b []byte) (SuffrageInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(SuffrageInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not SuffrageInfo; type=%T", i)
	} else {
		return v, nil
	}
}
