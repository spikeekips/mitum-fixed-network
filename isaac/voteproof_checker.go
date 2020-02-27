package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/logging"
)

var (
	IgnoreVoteproofError = errors.NewError("Voteproof should be ignored")
)

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

type VoteproofValidationChecker struct {
	*logging.Logger
	lastBlock         Block
	lastINITVoteproof Voteproof
	voteproof         Voteproof
	css               *ConsensusStates
}

func (vpc *VoteproofValidationChecker) CheckHeight() (bool, error) {
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

func (vpc *VoteproofValidationChecker) CheckINITVoteproof() (bool, error) {
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

func (vpc *VoteproofValidationChecker) CheckACCEPTVoteproof() (bool, error) {
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

// TODO check, signer is inside suffrage
// TODO check, signer of VoteproofNodeFact is valid Ballot.Signer()

var StopBootingError = errors.NewError("stop booting process")

type VoteproofBootingChecker struct {
	*logging.Logger
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
