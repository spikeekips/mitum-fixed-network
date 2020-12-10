package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type StateBootingHandler struct {
	*BaseStateHandler
	suffrage base.Suffrage
}

func NewStateBootingHandler(
	local *Local,
	suffrage base.Suffrage,
) (*StateBootingHandler, error) {
	cs := &StateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(local, nil, base.StateBooting),
		suffrage:         suffrage,
	}
	cs.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-booting-handler")
	})

	return cs, nil
}

func (cs *StateBootingHandler) Activate(_ *StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	cs.Log().Debug().Msg("activated")

	cs.activate()

	return nil
}

func (cs *StateBootingHandler) activate() {
	cs.BaseStateHandler.activate()

	go func() {
		var ctx *StateChangeContext
		if c, err := cs.initialize(); err != nil {
			cs.Log().Error().Err(err).Msg("failed to initialize at booting")

			return
		} else if c != nil {
			ctx = c
		}

		if ctx != nil {
			go func() {
				if err := cs.ChangeState(ctx.To(), ctx.Voteproof(), ctx.Ballot()); err != nil {
					cs.Log().Error().Err(err).Msg("ChangeState error")
				}
			}()
		}
	}()
}

func (cs *StateBootingHandler) Deactivate(_ *StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	cs.deactivate()

	cs.Log().Debug().Msg("deactivated")

	return nil
}

func (cs *StateBootingHandler) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, cs.Log())
	l.Debug().Msg("got Seal")

	return nil
}

func (cs *StateBootingHandler) NewVoteproof(voteproof base.Voteproof) error {
	l := loggerWithVoteproofID(voteproof, cs.Log())

	l.Debug().Msg("got Voteproof")

	return nil
}

func (cs *StateBootingHandler) initialize() (*StateChangeContext, error) {
	cs.Log().Debug().Msg("trying to initialize")

	if err := cs.checkBlock(); err != nil {
		cs.Log().Error().Err(err).Msg("something wrong to check blocks")

		if storage.IsNotFoundError(err) {
			if ctx, err0 := cs.whenEmptyBlocks(); err0 != nil {
				return nil, err0
			} else if ctx != nil {
				return ctx, nil
			}

			return nil, nil
		}

		return nil, err
	}

	cs.Log().Debug().Msg("initialized; moves to joining")

	return NewStateChangeContext(base.StateBooting, base.StateJoining, nil, nil), nil
}

func (cs *StateBootingHandler) checkBlock() error {
	cs.Log().Debug().Msg("trying to check block")
	defer cs.Log().Debug().Msg("checked block")

	var blk block.Block
	switch b, err := storage.CheckBlockEmpty(cs.local.Storage(), cs.local.BlockFS()); {
	case err != nil:
		return err
	case b == nil:
		return storage.NotFoundError.Errorf("empty block")
	default:
		blk = b
	}

	if err := blk.IsValid(cs.local.Policy().NetworkID()); err != nil {
		return xerrors.Errorf("invalid block found, clean up block: %w", err)
	} else {
		cs.Log().Debug().Hinted("block", blk.Manifest()).Msg("valid initial block found")
	}

	return nil
}

func (cs *StateBootingHandler) whenEmptyBlocks() (*StateChangeContext, error) {
	// NOTE clean storages
	if err := storage.Clean(cs.local.Storage(), cs.local.BlockFS(), false); err != nil {
		return nil, err
	}

	if len(cs.suffrage.Nodes()) < 2 { // NOTE suffrage nodes has local node itself
		return nil, xerrors.Errorf("empty block, but no other nodes; can not sync")
	}

	return NewStateChangeContext(base.StateBooting, base.StateSyncing, nil, nil), nil
}
