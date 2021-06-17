package block

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeManifest(b []byte, enc encoder.Encoder) (Manifest, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Manifest); !ok {
		return nil, util.WrongTypeError.Errorf("not Manifest; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeConsensusInfo(b []byte, enc encoder.Encoder) (ConsensusInfo, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(ConsensusInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not ConsensusInfoifest; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeSuffrageInfo(b []byte, enc encoder.Encoder) (SuffrageInfo, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(SuffrageInfo); !ok {
		return nil, util.WrongTypeError.Errorf("not SuffrageInfo; type=%T", i)
	} else {
		return v, nil
	}
}
