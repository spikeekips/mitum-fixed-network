package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type BlockConsensusInfoV0 struct {
	initVoteproof   base.Voteproof
	acceptVoteproof base.Voteproof
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
	return util.ConcatBytesSlice(
		bc.initVoteproof.Bytes(),
		bc.acceptVoteproof.Bytes(),
	)
}

func (bc BlockConsensusInfoV0) INITVoteproof() base.Voteproof {
	return bc.initVoteproof
}

func (bc BlockConsensusInfoV0) ACCEPTVoteproof() base.Voteproof {
	return bc.acceptVoteproof
}
