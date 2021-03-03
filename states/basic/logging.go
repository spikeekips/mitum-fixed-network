package basicstates

import "github.com/spikeekips/mitum/util/logging"

func LoggerWithStateSwitchContext(sctx StateSwitchContext, l logging.Logger) logging.Logger {
	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		var voteproofID string
		if sctx.Voteproof() != nil {
			voteproofID = sctx.Voteproof().ID()
		}

		return ctx.Dict("state_context", logging.Dict().
			Str("from", sctx.FromState().String()).
			Str("to", sctx.ToState().String()).
			Interface("error", sctx.Err()).
			Str("voteproof", voteproofID),
		).(logging.Context)
	})
}
