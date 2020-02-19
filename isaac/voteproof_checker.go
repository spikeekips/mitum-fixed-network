package isaac

import (
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/logging"
	"golang.org/x/xerrors"
)

var (
	IgnoreVoteProofError = errors.NewError("VoteProof should be ignored")
)

type ConsensusStateToBeChangeError struct {
	errors.CError
	FromState ConsensusState
	ToState   ConsensusState
	VoteProof VoteProof
}

func (ce ConsensusStateToBeChangeError) Error() string {
	return ce.CError.Error()
}

func (ce ConsensusStateToBeChangeError) ConsensusStateChangeContext() ConsensusStateChangeContext {
	return NewConsensusStateChangeContext(ce.FromState, ce.ToState, ce.VoteProof)
}

func NewConsensusStateToBeChangeError(
	fromState, toState ConsensusState, voteProof VoteProof,
) ConsensusStateToBeChangeError {
	return ConsensusStateToBeChangeError{
		CError:    errors.NewError("ConsensusState needs to be changed"),
		FromState: fromState,
		ToState:   toState,
		VoteProof: voteProof,
	}
}

type VoteProofValidationChecker struct {
	*logging.Logger
	lastBlock         Block
	lastINITVoteProof VoteProof
	voteProof         VoteProof
	css               *ConsensusStates
}

func (vpc *VoteProofValidationChecker) CheckHeight() (bool, error) {
	l := loggerWithVoteProof(vpc.voteProof, vpc.Log())

	d := vpc.voteProof.Height() - (vpc.lastBlock.Height() + 1)

	if d > 0 {
		l.Debug().
			Int64("local_block_height", vpc.lastBlock.Height().Int64()).
			Msg("VoteProof has higher height from local block")

		var fromState ConsensusState
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewConsensusStateToBeChangeError(fromState, ConsensusStateSyncing, vpc.voteProof)
	}

	if d < 0 {
		l.Debug().
			Int64("local_block_height", vpc.lastBlock.Height().Int64()).
			Msg("VoteProof has lower height from local block; ignore it")

		return false, IgnoreVoteProofError
	}

	return true, nil
}

func (vpc *VoteProofValidationChecker) CheckINITVoteProof() (bool, error) {
	if vpc.voteProof.Stage() != StageINIT {
		return true, nil
	}

	l := loggerWithVoteProof(vpc.voteProof, vpc.Log())

	if err := checkBlockWithINITVoteProof(vpc.lastBlock, vpc.voteProof); err != nil {
		l.Error().Err(err).Send()

		var fromState ConsensusState
		if vpc.css.ActiveHandler() != nil {
			fromState = vpc.css.ActiveHandler().State()
		}

		return false, NewConsensusStateToBeChangeError(fromState, ConsensusStateSyncing, vpc.voteProof)
	}

	return true, nil
}

func (vpc *VoteProofValidationChecker) CheckACCEPTVoteProof() (bool, error) {
	if vpc.voteProof.Stage() != StageACCEPT {
		return true, nil
	}

	if vpc.lastINITVoteProof.Round() != vpc.voteProof.Round() {
		return false, xerrors.Errorf("VoteProof has different round from last init voteproof: voteproof=%d last=%d",
			vpc.voteProof.Round(), vpc.lastINITVoteProof.Round(),
		)
	}

	return true, nil
}

// TODO check, signer is inside suffrage
// TODO check, signer of VoteProofNodeFact is valid Ballot.Signer()

var StopBootingError = errors.NewError("stop booting process")

type VoteProofBootingChecker struct {
	*logging.Logger
	localState      *LocalState
	lastBlock       Block
	initVoteProof   VoteProof
	acceptVoteProof VoteProof
}

func NewVoteProofBootingChecker(localState *LocalState) (*VoteProofBootingChecker, error) {
	initVoteProof := localState.LastINITVoteProof()
	if err := initVoteProof.IsValid(nil); err != nil {
		return nil, err
	}

	acceptVoteProof := localState.LastACCEPTVoteProof()
	if err := acceptVoteProof.IsValid(nil); err != nil {
		return nil, err
	}

	return &VoteProofBootingChecker{
		lastBlock:       localState.LastBlock(),
		initVoteProof:   initVoteProof,
		acceptVoteProof: acceptVoteProof,
	}, nil
}

func (vpc *VoteProofBootingChecker) CheckACCEPTVoteProofHeight() (bool, error) {
	switch d := vpc.acceptVoteProof.Height() - vpc.lastBlock.Height(); {
	case d == 0:
	default:
		// wrong ACCEPTVoteProof of last block, something wrong
		return false, StopBootingError.Wrapf(
			"missing ACCEPTVoteProof found: voteProof.Height()=%d != block.Height()=%d",
			vpc.acceptVoteProof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.acceptVoteProof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Wrapf(
			"round of ACCEPTVoteProof of same height not matched: voteProof.Round()=%d block.Round()=%d",
			vpc.acceptVoteProof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.acceptVoteProof.Majority().(ACCEPTBallotFact)
	if !vpc.lastBlock.Hash().Equal(fact.NewBlock()) {
		return false, StopBootingError.Wrapf(
			"block hash of ACCEPTVoteProof of same height not matched: voteProof.Block()=%s block.Block()=%s",
			fact.NewBlock(), vpc.lastBlock.Hash(),
		)
	}

	return true, nil
}

func (vpc *VoteProofBootingChecker) CheckINITVoteProofHeight() (bool, error) {
	switch d := vpc.initVoteProof.Height() - vpc.lastBlock.Height(); {
	case d == 0:
	default:
		return false, StopBootingError.Wrapf(
			"missing INITVoteProof found: voteProof.Height()=%d != block.Height()=%d",
			vpc.initVoteProof.Height(), vpc.lastBlock.Height(),
		)
	}

	if vpc.initVoteProof.Round() != vpc.lastBlock.Round() {
		return false, StopBootingError.Wrapf(
			"round of INITVoteProof of same height not matched: voteProof.Round()=%d block.Round()=%d",
			vpc.initVoteProof.Round(), vpc.lastBlock.Round(),
		)
	}

	fact := vpc.initVoteProof.Majority().(INITBallotFact)
	if !vpc.lastBlock.PreviousBlock().Equal(fact.PreviousBlock()) {
		return false, StopBootingError.Wrapf(
			"previous block hash of INITVoteProof of same height not matched: voteProof.Block()=%s block.Block()=%s",
			fact.PreviousBlock(), vpc.lastBlock.Hash(),
		)
	}

	return true, nil
}
