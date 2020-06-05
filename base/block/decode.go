package block

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func decodeManifest(enc encoder.Encoder, b []byte) (Manifest, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Manifest); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Manifest; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeBlockConsensusInfo(enc encoder.Encoder, b []byte) (BlockConsensusInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(BlockConsensusInfo); !ok {
		return nil, hint.InvalidTypeError.Errorf("not ConsensusInfoifest; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeSuffrageInfo(enc encoder.Encoder, b []byte) (SuffrageInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(SuffrageInfo); !ok {
		return nil, hint.InvalidTypeError.Errorf("not SuffrageInfo; type=%T", i)
	} else {
		return v, nil
	}
}
