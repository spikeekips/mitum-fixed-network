package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
)

type StateBrokenHandler struct {
	*BaseStateHandler
}

func NewStateBrokenHandler(localstate *Localstate) (*StateBrokenHandler, error) {
	ss := &StateBrokenHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, nil, StateBroken),
	}
	ss.BaseStateHandler.Logging = logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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

func (ss *StateBrokenHandler) NewVoteproof(Voteproof) error {
	return nil
}
