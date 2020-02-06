package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/xerrors"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logger
	localState *LocalState
	ballotbox  *Ballotbox
	suffrage   Suffrage
	states     map[ConsensusState]ConsensusStateHandler
	activated  ConsensusStateHandler
}

func NewConsensusStates(
	localState *LocalState,
	ballotbox *Ballotbox,
	suffrage Suffrage,
	joining *ConsensusStateJoiningHandler,
	consensus *ConsensusStateConsensusHandler,
	syncing ConsensusStateHandler,
	broken ConsensusStateHandler,
) *ConsensusStates {
	return &ConsensusStates{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-states")
		}),
		localState: localState,
		ballotbox:  ballotbox,
		suffrage:   suffrage,
		states: map[ConsensusState]ConsensusStateHandler{
			ConsensusStateJoining:   joining,
			ConsensusStateConsensus: consensus,
			ConsensusStateSyncing:   syncing,
			ConsensusStateBroken:    broken,
		},
	}
}

// Activated returns the current activated handler.
func (css *ConsensusStates) Activated() ConsensusStateHandler {
	css.RLock()
	defer css.RUnlock()

	return css.activated
}

func (css *ConsensusStates) newVoteProof(vp VoteProof) error {
	return css.Activated().NewVoteProof(vp)
}

// NewSeal receives Seal and hand it over to handler;
// - Seal is considered it should be already checked IsValid().
// - if Seal is signed by LocalNode, it will be ignored.
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	if css.Activated() == nil {
		return xerrors.Errorf("no activated handler")
	}

	l := loggerWithSeal(sl, css.Log()).With().
		Str("handler", css.Activated().State().String()).
		Logger()

	if sl.Signer().Equal(css.localState.Node().Publickey()) {
		err := xerrors.Errorf("Seal is from LocalNode")
		l.Error().Err(err).Send()

		return err
	}

	// TODO check validation for Seal
	if err := css.validateSeal(sl); err != nil {
		l.Error().Err(err).Msg("seal validation failed")

		return err
	}

	go func() {
		if err := css.Activated().NewSeal(sl); err != nil {
			l.Error().
				Err(err).Msg("activated handler can not receive Seal")
		}
	}()

	b, isBallot := sl.(Ballot)
	if !isBallot {
		return nil
	}

	switch b.Stage() {
	case StageINIT, StageACCEPT:
		return css.vote(b)
	}

	return nil
}

func (css *ConsensusStates) validateSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case Proposal:
		return css.validateProposal(t)
	case Ballot:
		return css.validateBallot(t)
	}

	return nil
}

func (css *ConsensusStates) validateBallot(_ Ballot) error {
	// TODO check validation
	// - Ballot.Node() is in suffrage
	// - Ballot.Height() is equal or higher than LastINITVoteProof.
	// - Ballot.Round() is equal or higher than LastINITVoteProof.
	return nil
}

func (css *ConsensusStates) validateProposal(proposal Proposal) error {
	// TODO Proposal should be validated by ConsensusStates.

	l := loggerWithBallot(proposal, css.Log())

	// TODO check Proposer is valid proposer
	if !css.suffrage.IsProposer(proposal.Height(), proposal.Round(), proposal.Node()) {
		err := xerrors.Errorf(
			"wrong proposer; height=%d round=%d, but proposer=%v",
			proposal.Height(),
			proposal.Round(),
			proposal.Node(),
		)

		l.Error().Err(err).Msg("wrong proposer found")

		return err
	}

	return nil
}

func (css *ConsensusStates) vote(ballot Ballot) error {
	vp, err := css.ballotbox.Vote(ballot)
	if err != nil {
		return err
	} else if !vp.IsFinished() {
		return nil
	}

	return css.newVoteProof(vp)
}
