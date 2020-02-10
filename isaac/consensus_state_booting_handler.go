package isaac

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
)

type ConsensusStateBootingHandler struct {
	*BaseStateHandler
}

func NewConsensusStateBootingHandler(
	localState *LocalState,
) (*ConsensusStateBootingHandler, error) {
	if lastBlock := localState.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &ConsensusStateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(localState, ConsensusStateBooting),
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-booting-handler")
	})

	return cs, nil
}

func (cs *ConsensusStateBootingHandler) Activate(ctx ConsensusStateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("activated")

	go func() {
		if err := cs.check(); err != nil {
			cs.Log().Error().Err(err).Msg("failed to check")
		}
	}()

	return nil
}

func (cs *ConsensusStateBootingHandler) Deactivate(ctx ConsensusStateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("deactivated")

	return nil
}

// NewSeal only cares on INIT ballot and it's VoteProof.
func (cs *ConsensusStateBootingHandler) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, cs.Log())
	l.Debug().Msg("got Seal")

	return nil
}

// NewVoteProof receives VoteProof. If received, stop broadcasting INIT ballot.
func (cs *ConsensusStateBootingHandler) NewVoteProof(vp VoteProof) error {
	l := loggerWithVoteProof(vp, cs.Log())

	l.Debug().Msg("got VoteProof")

	return nil
}

func (cs *ConsensusStateBootingHandler) check() error {
	cs.Log().Debug().Msg("trying to check")

	// TODO set Policies

	cs.Log().Debug().Msg("checked; moves to joining")

	if err := cs.checkBlock(); err != nil {
		xerr := errors.NewError("failed to check block").Wrap(err).(errors.Error)
		cs.Log().Error().Err(xerr).Send()

		if errc := cs.ChangeState(ConsensusStateSyncing, nil); errc != nil {
			cs.Log().Error().Err(errc).Send()
			return xerr.Wrap(errc)
		}

		return xerr
	}

	if err := cs.checkVoteProof(); err != nil {
		xerr := errors.NewError("failed to check VoteProof").Wrap(err).(errors.Error)
		cs.Log().Error().Err(xerr).Send()

		if errc := cs.ChangeState(ConsensusStateSyncing, nil); errc != nil {
			cs.Log().Error().Err(errc).Send()
			return xerr.Wrap(errc)
		}

		return xerr
	}

	if err := cs.ChangeState(ConsensusStateJoining, nil); err != nil {
		cs.Log().Error().Err(err).Send()
	}

	return nil
}

func (cs *ConsensusStateBootingHandler) checkBlock() error {
	block := cs.localState.LastBlock()
	if block == nil {
		return xerrors.Errorf("empty Block")
	}

	return nil
}

func (cs *ConsensusStateBootingHandler) checkVoteProof() error {
	ivp := cs.localState.LastINITVoteProof()
	if ivp == nil {
		return xerrors.Errorf("empty INIT VoteProof")
	} else if err := ivp.IsValid(nil); err != nil {
		return err
	}

	avp := cs.localState.LastACCEPTVoteProof()
	if avp == nil {
		return xerrors.Errorf("empty ACCEPT VoteProof")
	} else if err := avp.IsValid(nil); err != nil {
		return err
	}

	block := cs.localState.LastBlock()

	if err := avp.CompareWithBlock(block); err != nil {
		return err
	}

	if err := ivp.CompareWithBlock(block); err != nil {
		return err
	}

	return nil
}
