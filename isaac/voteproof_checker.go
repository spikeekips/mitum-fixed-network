package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	IgnoreVoteproofError = errors.NewError("Voteproof should be ignored")
	StopBootingError     = errors.NewError("stop booting process")
)

type StateToBeChangeError struct {
	*errors.NError
	FromState base.State
	ToState   base.State
	Voteproof base.Voteproof
	Ballot    ballot.Ballot
}

func (ce *StateToBeChangeError) StateChangeContext() StateChangeContext {
	return NewStateChangeContext(ce.FromState, ce.ToState, ce.Voteproof, ce.Ballot)
}

func NewStateToBeChangeError(
	fromState, toState base.State, voteproof base.Voteproof, blt ballot.Ballot,
) *StateToBeChangeError {
	return &StateToBeChangeError{
		NError:    errors.NewError("State needs to be changed"),
		FromState: fromState,
		ToState:   toState,
		Voteproof: voteproof,
		Ballot:    blt,
	}
}

type VoteProofChecker struct {
	*logging.Logging
	voteproof  base.Voteproof
	suffrage   base.Suffrage
	localstate *Localstate
}

// NOTE VoteProofChecker should check the signer of VoteproofNodeFact is valid
// Ballot.Signer(), but it takes a little bit time to gather the Ballots from
// the other node, so this will be ignored at this time for performance reason.

func NewVoteProofChecker(voteproof base.Voteproof, localstate *Localstate, suffrage base.Suffrage) *VoteProofChecker {
	return &VoteProofChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "voteproof-checker")
		}),
		voteproof:  voteproof,
		suffrage:   suffrage,
		localstate: localstate,
	}
}

func (vc *VoteProofChecker) CheckIsValid() (bool, error) {
	networkID := vc.localstate.Policy().NetworkID()
	if err := vc.voteproof.IsValid(networkID); err != nil {
		return false, err
	}

	return true, nil
}

func (vc *VoteProofChecker) CheckNodeIsInSuffrage() (bool, error) {
	for n := range vc.voteproof.Ballots() {
		if !vc.suffrage.IsInside(n) {
			vc.Log().Debug().Str("node", n.String()).Msg("voteproof has the vote from unknown node")
			return false, nil
		}
	}

	return true, nil
}

// TODO CheckThreshold checks Threshold in Voteproof should be checked whether
// it has correct value at that block height.

func (vc *VoteProofChecker) CheckThreshold() (bool, error) {
	threshold := vc.localstate.Policy().Threshold()
	if !threshold.Equal(vc.voteproof.Threshold()) {
		vc.Log().Debug().
			Interface("threshold", vc.voteproof.Threshold()).
			Interface("expected", threshold).
			Msg("voteproof has different threshold")
		return false, nil
	}

	return true, nil
}

type VoteproofConsensusStateChecker struct {
	*logging.Logging
	lastBlock         block.Block
	lastINITVoteproof base.Voteproof
	voteproof         base.Voteproof
	css               *ConsensusStates
}

func NewVoteproofConsensusStateChecker(
	lastBlock block.Block,
	lastINITVoteproof base.Voteproof,
	voteproof base.Voteproof,
	css *ConsensusStates,
) *VoteproofConsensusStateChecker {
	return &VoteproofConsensusStateChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "voteproof-validation-checker")
		}),
		lastBlock:         lastBlock,
		lastINITVoteproof: lastINITVoteproof,
		voteproof:         voteproof,
		css:               css,
	}
}

func (vpc *VoteproofConsensusStateChecker) CheckHeight() (bool, error) {
	// TODO reduce the duplicated voteproof.
	l := loggerWithVoteproof(vpc.voteproof, vpc.Log())

	d := vpc.voteproof.Height() - (vpc.lastBlock.Height() + 1)

	if d > 0 {
		l.Debug().
			Hinted("local_block_height", vpc.lastBlock.Height()).
			Msg("Voteproof has higher height from local block")

		var fromState base.State
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewStateToBeChangeError(fromState, base.StateSyncing, vpc.voteproof, nil)
	}

	if d < 0 {
		l.Debug().
			Hinted("local_block_height", vpc.lastBlock.Height()).
			Msg("Voteproof has lower height from local block; ignore it")

		return false, IgnoreVoteproofError
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckINITVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != base.StageINIT {
		return true, nil
	}

	l := loggerWithVoteproof(vpc.voteproof, vpc.Log())

	if err := checkBlockWithINITVoteproof(vpc.lastBlock, vpc.voteproof); err != nil {
		l.Error().Err(err).Msg("invalid init voteproof")

		var fromState base.State
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewStateToBeChangeError(fromState, base.StateSyncing, vpc.voteproof, nil)
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckACCEPTVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != base.StageACCEPT {
		return true, nil
	}

	if vpc.lastINITVoteproof.Round() != vpc.voteproof.Round() {
		return false, xerrors.Errorf("Voteproof has different round from last init voteproof: voteproof=%d last=%d",
			vpc.voteproof.Round(), vpc.lastINITVoteproof.Round(),
		)
	}

	return true, nil
}

type VoteproofBootingChecker struct {
	*logging.Logging
	localstate      *Localstate // nolint
	lastBlock       block.Block
	initVoteproof   base.Voteproof // NOTE these Voteproof are from last block
	acceptVoteproof base.Voteproof
}

func NewVoteproofBootingChecker(localstate *Localstate) (*VoteproofBootingChecker, error) {
	initVoteproof := localstate.LastINITVoteproof()
	if err := initVoteproof.IsValid(nil); err != nil {
		return nil, err
	}

	acceptVoteproof := localstate.LastACCEPTVoteproof()
	if err := acceptVoteproof.IsValid(nil); err != nil {
		return nil, err
	}

	return &VoteproofBootingChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "voteproof-booting-checker")
		}),
		lastBlock:       localstate.LastBlock(),
		initVoteproof:   initVoteproof,
		acceptVoteproof: acceptVoteproof,
	}, nil
}

func (vpc *VoteproofBootingChecker) CheckACCEPTVoteproofHeight() (bool, error) {
	switch d := vpc.acceptVoteproof.Height() - vpc.lastBlock.Height(); {
	case d == 0:
	default:
		// TODO needs self-correction by syncing
		// wrong ACCEPTVoteproof of last block, something wrong
		return false, StopBootingError.Errorf(
			"missing ACCEPTVoteproof found: voteproof.Height()=%d != block.Height()=%d",
			vpc.acceptVoteproof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.acceptVoteproof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Errorf(
			"round of ACCEPTVoteproof of same height not matched: voteproof.Round()=%d block.Round()=%d",
			vpc.acceptVoteproof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.acceptVoteproof.Majority().(ballot.ACCEPTBallotFact)
	if !vpc.lastBlock.Hash().Equal(fact.NewBlock()) {
		return false, StopBootingError.Errorf(
			"block hash of ACCEPTVoteproof of same height not matched: voteproof.Block()=%s block.Block()=%s",
			fact.NewBlock(), vpc.lastBlock.Hash(),
		)
	}

	return true, nil
}

func (vpc *VoteproofBootingChecker) CheckINITVoteproofHeight() (bool, error) {
	switch d := vpc.initVoteproof.Height() - vpc.lastBlock.Height(); {
	case d == 0:
	default:
		return false, StopBootingError.Errorf(
			"missing INITVoteproof found: voteproof.Height()=%d != block.Height()=%d",
			vpc.initVoteproof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.initVoteproof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Errorf(
			"round of INITVoteproof of same height not matched: voteproof.Round()=%d block.Round()=%d",
			vpc.initVoteproof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.initVoteproof.Majority().(ballot.INITBallotFact)
	if !vpc.lastBlock.PreviousBlock().Equal(fact.PreviousBlock()) {
		return false, StopBootingError.Errorf(
			"previous block hash of INITVoteproof of same height not matched: voteproof.Block()=%s block.Block()=%s",
			fact.PreviousBlock(), vpc.lastBlock.Hash(),
		)
	}

	return true, nil
}
