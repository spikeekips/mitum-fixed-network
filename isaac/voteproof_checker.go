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

type VoteProofChecker struct {
	*logging.Logger
	lastBlock         Block
	lastINITVoteProof VoteProof
	voteProof         VoteProof
	css               *ConsensusStates
}

func (vpc *VoteProofChecker) CheckHeight() (bool, error) {
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

func (vpc *VoteProofChecker) CheckINITVoteProof() (bool, error) {
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

func (vpc *VoteProofChecker) CheckACCEPTVoteProof() (bool, error) {
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

type VoteProofValidationChecker struct {
	*logging.Logger
	voteProof VoteProof
	b         []byte
}

func (vpc *VoteProofValidationChecker) CheckValidate() (bool, error) {
	if err := vpc.voteProof.IsValid(vpc.b); err != nil {
		return false, err
	}

	return true, nil
}

// TODO check, signer is inside suffrage
// TODO check, signer of VoteProofNodeFact is valid Ballot.Signer()
