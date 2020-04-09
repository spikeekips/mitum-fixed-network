package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

type StateBrokenHandler struct {
	*BaseStateHandler
}

func NewStateBrokenHandler(localstate *Localstate) (*StateBrokenHandler, error) {
	ss := &StateBrokenHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, nil, base.StateBroken),
	}
	ss.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-broken-handler")
	})

	return ss, nil
}

func (ss *StateBrokenHandler) Activate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, ss.Log())
	l.Debug().Msg("activated")

	return nil
}

func (ss *StateBrokenHandler) Deactivate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, ss.Log())
	l.Debug().Msg("deactivated")

	return nil
}

func (ss *StateBrokenHandler) NewSeal(seal.Seal) error {
	return nil
}

func (ss *StateBrokenHandler) NewVoteproof(base.Voteproof) error {
	return nil
}
