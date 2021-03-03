package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/logging"
)

type VoteProofChecker struct {
	*logging.Logging
	voteproof base.Voteproof
	policy    *LocalPolicy
	suffrage  base.Suffrage
}

// NOTE VoteProofChecker should check the signer of VoteproofNodeFact is valid
// Ballot.Signer(), but it takes a little bit time to gather the Ballots from
// the other node, so this will be ignored at this time for performance reason.

func NewVoteProofChecker(voteproof base.Voteproof, policy *LocalPolicy, suffrage base.Suffrage) *VoteProofChecker {
	return &VoteProofChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "voteproof-checker")
		}),
		voteproof: voteproof,
		policy:    policy,
		suffrage:  suffrage,
	}
}

func (vc *VoteProofChecker) IsValid() (bool, error) {
	networkID := vc.policy.NetworkID()
	if err := vc.voteproof.IsValid(networkID); err != nil {
		return false, err
	}

	return true, nil
}

func (vc *VoteProofChecker) NodeIsInSuffrage() (bool, error) {
	for i := range vc.voteproof.Votes() {
		nf := vc.voteproof.Votes()[i]
		if !vc.suffrage.IsInside(nf.Node()) {
			vc.Log().Debug().Str("node", nf.Node().String()).Msg("voteproof has the vote from unknown node")

			return false, nil
		}
	}

	return true, nil
}

// CheckThreshold checks Threshold only for new incoming Voteproof.
func (vc *VoteProofChecker) CheckThreshold() (bool, error) {
	tr := vc.policy.ThresholdRatio()
	if tr != vc.voteproof.ThresholdRatio() {
		vc.Log().Debug().
			Interface("threshold_ratio", vc.voteproof.ThresholdRatio()).
			Interface("expected", tr).
			Msg("voteproof has different threshold ratio")
		return false, nil
	}

	return true, nil
}
