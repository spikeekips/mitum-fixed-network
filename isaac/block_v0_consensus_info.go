package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
)

type BlockConsensusInfoV0 struct {
	initVoteproof   Voteproof
	acceptVoteproof Voteproof
}

func (bc BlockConsensusInfoV0) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		bc.initVoteproof,
		bc.acceptVoteproof,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (bc BlockConsensusInfoV0) Hint() hint.Hint {
	return BlockConsensusInfoV0Hint
}

func (bc BlockConsensusInfoV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		bc.initVoteproof.Bytes(),
		bc.acceptVoteproof.Bytes(),
	})
}

func (bc BlockConsensusInfoV0) INITVoteproof() Voteproof {
	return bc.initVoteproof
}

func (bc BlockConsensusInfoV0) ACCEPTVoteproof() Voteproof {
	return bc.acceptVoteproof
}
