package isaac

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type ConsensusStateBootingHandler struct {
	*BaseStateHandler
}

func NewConsensusStateBootingHandler(
	localState *LocalState,
	proposalProcessor ProposalProcessor,
) (*ConsensusStateBootingHandler, error) {
	if lastBlock := localState.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &ConsensusStateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(localState, proposalProcessor, ConsensusStateBooting),
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
		if err := cs.initialize(); err != nil {
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

func (cs *ConsensusStateBootingHandler) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, cs.Log())
	l.Debug().Msg("got Seal")

	return nil
}

func (cs *ConsensusStateBootingHandler) NewVoteProof(vp VoteProof) error {
	l := loggerWithVoteProof(vp, cs.Log())

	l.Debug().Msg("got VoteProof")

	return nil
}

func (cs *ConsensusStateBootingHandler) initialize() error {
	cs.Log().Debug().Msg("trying to initialize")

	if err := cs.check(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("initialized; moves to joining")

	return cs.ChangeState(ConsensusStateJoining, nil)
}

func (cs *ConsensusStateBootingHandler) check() error {
	cs.Log().Debug().Msg("trying to check")
	defer cs.Log().Debug().Msg("complete to check")

	if err := cs.checkBlock(); err != nil {
		cs.Log().Error().Err(err).Send()

		if err0 := cs.ChangeState(ConsensusStateSyncing, nil); err0 != nil {
			// TODO wrap err
			return err0
		}

		return err
	}

	if err := cs.checkVoteProof(); err != nil {
		cs.Log().Error().Err(err).Send()

		var ctx ConsensusStateToBeChangeError
		if xerrors.As(err, &ctx) {
			if err0 := cs.ChangeState(ctx.ToState, ctx.VoteProof); err0 != nil {
				// TODO wrap err
				return err0
			}

			return nil
		} else if xerrors.Is(err, StopBootingError) {
			return err
		}

	}

	return nil
}

func (cs *ConsensusStateBootingHandler) checkBlock() error {
	cs.Log().Debug().Msg("trying to check block")
	defer cs.Log().Debug().Msg("complete to check block")

	block := cs.localState.LastBlock()
	if block == nil {
		return xerrors.Errorf("empty Block")
	} else if err := block.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateBootingHandler) checkVoteProof() error {
	cs.Log().Debug().Msg("trying to check VoteProofs")
	defer cs.Log().Debug().Msg("trying to check VoteProofs")

	vpc, err := NewVoteProofBootingChecker(cs.localState)
	if err != nil {
		return err
	}

	return util.NewChecker("voteproof-booting-checker", []util.CheckerFunc{
		vpc.CheckACCEPTVoteProofHeight,
		vpc.CheckINITVoteProofHeight,
	}).Check()
}
