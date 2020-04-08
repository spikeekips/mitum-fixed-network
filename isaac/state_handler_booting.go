package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type StateBootingHandler struct {
	*BaseStateHandler
}

func NewStateBootingHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
) (*StateBootingHandler, error) {
	if lastBlock := localstate.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &StateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, StateBooting),
	}
	cs.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-booting-handler")
	})

	return cs, nil
}

func (cs *StateBootingHandler) Activate(ctx StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	l := loggerWithStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("activated")

	go func() {
		if err := cs.initialize(); err != nil {
			cs.Log().Error().Err(err).Msg("failed to check")
		}
	}()

	return nil
}

func (cs *StateBootingHandler) Deactivate(ctx StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	l := loggerWithStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("deactivated")

	return nil
}

func (cs *StateBootingHandler) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, cs.Log())
	l.Debug().Msg("got Seal")

	return nil
}

func (cs *StateBootingHandler) NewVoteproof(voteproof Voteproof) error {
	l := loggerWithVoteproof(voteproof, cs.Log())

	l.Debug().Msg("got Voteproof")

	return nil
}

func (cs *StateBootingHandler) initialize() error {
	cs.Log().Debug().Msg("trying to initialize")

	if err := cs.check(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("initialized; moves to joining")

	return cs.ChangeState(StateJoining, nil, nil)
}

func (cs *StateBootingHandler) check() error {
	cs.Log().Debug().Msg("trying to check")
	defer cs.Log().Debug().Msg("complete to check")

	if err := cs.checkBlock(); err != nil {
		cs.Log().Error().Err(err).Msg("checked block")

		// TODO syncing handler should support syncing without voteproof and ballot
		if err0 := cs.ChangeState(StateSyncing, nil, nil); err0 != nil {
			return xerrors.Errorf("failed to change state; %w", err0)
		}

		return err
	}

	if err := cs.checkVoteproof(); err != nil {
		cs.Log().Error().Err(err).Msg("checked voteproof")

		var ctx *StateToBeChangeError
		if xerrors.As(err, &ctx) {
			if err0 := cs.ChangeState(ctx.ToState, ctx.Voteproof, nil); err0 != nil {
				return xerrors.Errorf("failed to change state; %w", err0)
			}

			return nil
		} else if xerrors.Is(err, StopBootingError) {
			return err
		}
	}

	return nil
}

func (cs *StateBootingHandler) checkBlock() error {
	cs.Log().Debug().Msg("trying to check block")
	defer cs.Log().Debug().Msg("complete to check block")

	block := cs.localstate.LastBlock()
	if block == nil {
		return xerrors.Errorf("empty Block")
	} else if err := block.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cs *StateBootingHandler) checkVoteproof() error {
	cs.Log().Debug().Msg("trying to check Voteproofs")
	defer cs.Log().Debug().Msg("trying to check Voteproofs")

	vpc, err := NewVoteproofBootingChecker(cs.localstate)
	if err != nil {
		return err
	}
	_ = vpc.SetLogger(cs.Log())

	return util.NewChecker("voteproof-booting-checker", []util.CheckerFunc{
		vpc.CheckACCEPTVoteproofHeight,
		vpc.CheckINITVoteproofHeight,
	}).Check()
}
