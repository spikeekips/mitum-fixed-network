package isaac

import (
	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/logging"
)

var IgnoreVoteproofError = errors.NewError("Voteproof should be ignored")

type StateToBeChangeError struct {
	errors.CError
	FromState State
	ToState   State
	Voteproof Voteproof
}

func (ce StateToBeChangeError) Error() string {
	return ce.CError.Error()
}

func (ce StateToBeChangeError) StateChangeContext() StateChangeContext {
	return NewStateChangeContext(ce.FromState, ce.ToState, ce.Voteproof)
}

func NewStateToBeChangeError(
	fromState, toState State, voteproof Voteproof,
) StateToBeChangeError {
	return StateToBeChangeError{
		CError:    errors.NewError("State needs to be changed"),
		FromState: fromState,
		ToState:   toState,
		Voteproof: voteproof,
	}
}

type VoteProofChecker struct {
	*logging.Logging
	voteproof  Voteproof
	suffrage   Suffrage
	localstate *Localstate
}

// NOTE VoteProofChecker should check the signer of VoteproofNodeFact is valid
// Ballot.Signer(), but it takes a little bit time to gather the Ballots from
// the other node, so this will be ignored at this time for performance reason.

func NewVoteProofChecker(voteproof Voteproof, localstate *Localstate, suffrage Suffrage) *VoteProofChecker {
	return &VoteProofChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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
	lastBlock         Block
	lastINITVoteproof Voteproof
	voteproof         Voteproof
	css               *ConsensusStates
}

func NewVoteproofConsensusStateChecker(
	lastBlock Block,
	lastINITVoteproof Voteproof,
	voteproof Voteproof,
	css *ConsensusStates,
) *VoteproofConsensusStateChecker {
	return &VoteproofConsensusStateChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "voteproof-validation-checker")
		}),
		lastBlock:         lastBlock,
		lastINITVoteproof: lastINITVoteproof,
		voteproof:         voteproof,
		css:               css,
	}
}

func (vpc *VoteproofConsensusStateChecker) CheckHeight() (bool, error) {
	l := loggerWithVoteproof(vpc.voteproof, vpc.Log())

	d := vpc.voteproof.Height() - (vpc.lastBlock.Height() + 1)

	if d > 0 {
		l.Debug().
			Int64("local_block_height", vpc.lastBlock.Height().Int64()).
			Msg("Voteproof has higher height from local block")

		var fromState State
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewStateToBeChangeError(fromState, StateSyncing, vpc.voteproof)
	}

	if d < 0 {
		l.Debug().
			Int64("local_block_height", vpc.lastBlock.Height().Int64()).
			Msg("Voteproof has lower height from local block; ignore it")

		return false, IgnoreVoteproofError
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckINITVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != StageINIT {
		return true, nil
	}

	l := loggerWithVoteproof(vpc.voteproof, vpc.Log())

	if err := checkBlockWithINITVoteproof(vpc.lastBlock, vpc.voteproof); err != nil {
		l.Error().Err(err).Msg("invalid init voteproof")

		var fromState State
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewStateToBeChangeError(fromState, StateSyncing, vpc.voteproof)
	}

	return true, nil
}

func (vpc *VoteproofConsensusStateChecker) CheckACCEPTVoteproof() (bool, error) {
	if vpc.voteproof.Stage() != StageACCEPT {
		return true, nil
	}

	if vpc.lastINITVoteproof.Round() != vpc.voteproof.Round() {
		return false, xerrors.Errorf("Voteproof has different round from last init voteproof: voteproof=%d last=%d",
			vpc.voteproof.Round(), vpc.lastINITVoteproof.Round(),
		)
	}

	return true, nil
}

var StopBootingError = errors.NewError("stop booting process")

type VoteproofBootingChecker struct {
	*logging.Logging
	localstate      *Localstate // nolint
	lastBlock       Block
	initVoteproof   Voteproof // NOTE these Voteproof are from last block
	acceptVoteproof Voteproof
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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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
		return false, StopBootingError.Wrapf(
			"missing ACCEPTVoteproof found: voteproof.Height()=%d != block.Height()=%d",
			vpc.acceptVoteproof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.acceptVoteproof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Wrapf(
			"round of ACCEPTVoteproof of same height not matched: voteproof.Round()=%d block.Round()=%d",
			vpc.acceptVoteproof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.acceptVoteproof.Majority().(ACCEPTBallotFact)
	if !vpc.lastBlock.Hash().Equal(fact.NewBlock()) {
		return false, StopBootingError.Wrapf(
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
		return false, StopBootingError.Wrapf(
			"missing INITVoteproof found: voteproof.Height()=%d != block.Height()=%d",
			vpc.initVoteproof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.initVoteproof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Wrapf(
			"round of INITVoteproof of same height not matched: voteproof.Round()=%d block.Round()=%d",
			vpc.initVoteproof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.initVoteproof.Majority().(INITBallotFact)
	if !vpc.lastBlock.PreviousBlock().Equal(fact.PreviousBlock()) {
		return false, StopBootingError.Wrapf(
			"previous block hash of INITVoteproof of same height not matched: voteproof.Block()=%s block.Block()=%s",
			fact.PreviousBlock(), vpc.lastBlock.Hash(),
		)
	}

	return true, nil
}
