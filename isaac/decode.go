package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
)

func DecodeAddress(enc encoder.Encoder, b []byte) (Address, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Address); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Address; type=%T", i)
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
		return nil, hint.InvalidTypeError.Errorf("not Voteproof; type=%T", i)
	} else {
		return v, nil
	}
}

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
