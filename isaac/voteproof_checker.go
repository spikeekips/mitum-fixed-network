package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
)

var stateToBeChangeError = errors.NewError("State needs to be changed")

type StateToBeChangeError struct {
	*errors.NError
	ToState   base.State
	Voteproof base.Voteproof
	Ballot    ballot.Ballot
	Err       error
}

func NewStateToBeChangeError(
	toState base.State,
	voteproof base.Voteproof,
	blt ballot.Ballot,
	err error,
) *StateToBeChangeError {
	return &StateToBeChangeError{
		NError:    stateToBeChangeError,
		ToState:   toState,
		Voteproof: voteproof,
		Ballot:    blt,
		Err:       err,
	}
}

type VoteProofChecker struct {
	*logging.Logging
	voteproof base.Voteproof
	suffrage  base.Suffrage
	local     *Local
}

// NOTE VoteProofChecker should check the signer of VoteproofNodeFact is valid
// Ballot.Signer(), but it takes a little bit time to gather the Ballots from
// the other node, so this will be ignored at this time for performance reason.

func NewVoteProofChecker(voteproof base.Voteproof, local *Local, suffrage base.Suffrage) *VoteProofChecker {
	return &VoteProofChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "voteproof-checker")
		}),
		voteproof: voteproof,
		suffrage:  suffrage,
		local:     local,
	}
}

func (vc *VoteProofChecker) CheckIsValid() (bool, error) {
	networkID := vc.local.Policy().NetworkID()
	if err := vc.voteproof.IsValid(networkID); err != nil {
		return false, err
	}

	return true, nil
}

func (vc *VoteProofChecker) CheckNodeIsInSuffrage() (bool, error) {
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
	tr := vc.local.Policy().ThresholdRatio()
	if tr != vc.voteproof.ThresholdRatio() {
		vc.Log().Debug().
			Interface("threshold_ratio", vc.voteproof.ThresholdRatio()).
			Interface("expected", tr).
			Msg("voteproof has different threshold ratio")
		return false, nil
	}

	return true, nil
}

type VoteproofConsensusStateChecker struct {
	*logging.Logging
	lastManifest      block.Manifest
	lastINITVoteproof base.Voteproof
	voteproof         base.Voteproof
	css               *ConsensusStates
}

func NewVoteproofConsensusStateChecker(
	st storage.Storage,
	lastINITVoteproof base.Voteproof,
	voteproof base.Voteproof,
	css *ConsensusStates,
) (*VoteproofConsensusStateChecker, error) {
	var manifest block.Manifest
	if m, found, err := st.LastManifest(); err != nil {
		return nil, err
	} else if found {
		manifest = m
	}

	return &VoteproofConsensusStateChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			var li string
			if lastINITVoteproof != nil {
				li = lastINITVoteproof.ID()
			}
			return c.Str("module", "voteproof-validation-checker").
				Str("voteproof_id", voteproof.ID()).
				Str("last_init_voteproof_id", li)
		}),
		lastManifest:      manifest,
		lastINITVoteproof: lastINITVoteproof,
		voteproof:         voteproof,
		css:               css,
	}, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckHeight() (bool, error) {
	var height base.Height
	if vpc.lastManifest == nil {
		height = base.NilHeight
	} else {
		height = vpc.lastManifest.Height()
	}

	d := vpc.voteproof.Height() - (height + 1)

	if d > 0 {
		vpc.Log().Debug().
			Hinted("voteproof_height", vpc.voteproof.Height()).
			Hinted("local_block_height", height).
			Msg("Voteproof has higher height from local block")

		return false, NewStateToBeChangeError(
			base.StateSyncing, vpc.voteproof, nil,
			xerrors.Errorf("Voteproof has higher height from local block"),
		)
	}

	if d < 0 {
		vpc.Log().Debug().
			Hinted("local_block_height", height).
			Msg("Voteproof has lower height from local block; ignore it")

		return false, util.IgnoreError.Errorf("Voteproof has lower height from local block; ignore it")
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckINITVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != base.StageINIT {
		return true, nil
	}

	if err := checkBlockWithINITVoteproof(vpc.lastManifest, vpc.voteproof); err != nil {
		vpc.Log().Error().Err(err).Msg("werid init voteproof found")

		return false, NewStateToBeChangeError(base.StateSyncing, vpc.voteproof, nil, err)
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckACCEPTVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != base.StageACCEPT {
		return true, nil
	}

	ivp := vpc.lastINITVoteproof
	if ivp.Height() != vpc.voteproof.Height() || ivp.Round() != vpc.voteproof.Round() {
		vpc.Log().Debug().
			Hinted("last_init_voteproof_height", ivp.Height()).
			Hinted("last_init_voteproof_round", ivp.Round()).
			Hinted("voteproof_height", vpc.voteproof.Height()).
			Hinted("voteproof_round", vpc.voteproof.Round()).
			Msg("Voteproof has different round from last init voteproof")
	}

	return true, nil
}
