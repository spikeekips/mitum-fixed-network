package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (pr *ProposalV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	sls []valuehash.Hash,
	bVoteproof []byte,
) error {
	pr.BaseBallotV0 = bb
	pr.ProposalFactV0 = ProposalFactV0{
		BaseBallotFactV0: bf,
		seals:            sls,
	}

	if bVoteproof != nil {
		if i, err := base.DecodeVoteproof(enc, bVoteproof); err != nil {
			return err
		} else {
			pr.voteproof = i
		}
	}

	return nil
}
