package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

type StateBootingHandler struct {
	*BaseStateHandler
}

func NewStateBootingHandler(
	localstate *Localstate,
) (*StateBootingHandler, error) {
	cs := &StateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, nil, base.StateBooting),
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

func (cs *StateBootingHandler) NewVoteproof(voteproof base.Voteproof) error {
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

	return cs.ChangeState(base.StateJoining, nil, nil)
}

func (cs *StateBootingHandler) check() error {
	cs.Log().Debug().Msg("trying to check")
	defer cs.Log().Debug().Msg("checked")

	if err := cs.checkBlock(); err != nil {
		cs.Log().Error().Err(err).Msg("checked block")

		if xerrors.Is(err, storage.NotFoundError) {
			// TODO syncing handler should support syncing without voteproof and ballot
			if err0 := cs.ChangeState(base.StateSyncing, nil, nil); err0 != nil {
				return err0
			}
		}

		return err
	}

	return nil
}

func (cs *StateBootingHandler) checkBlock() error {
	cs.Log().Debug().Msg("trying to check block")
	defer cs.Log().Debug().Msg("checked block")

	var foundError error
	if blk, err := cs.localstate.Storage().LastBlock(); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			foundError = storage.NotFoundError.Errorf("empty Block")
		} else {
			foundError = err
		}
	} else if err := blk.IsValid(nil); err != nil {
		foundError = err
	} else {
		cs.Log().Debug().Hinted("block", blk.Manifest()).Msg("initial block found")
	}

	if foundError != nil {
		cs.Log().Debug().Err(foundError).Msg("initial block not found")
	}

	return foundError
}
