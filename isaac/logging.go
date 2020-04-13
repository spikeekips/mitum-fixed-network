package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

func loggerWithSeal(sl seal.Seal, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("seal_hash", sl.Hash()).(logging.Context).
			CallerWithSkipFrameCount(3)
	})

	var event logging.Emitter
	if ls, ok := sl.(logging.LogHintedMarshaler); ok {
		event = ll.Debug().HintedVerbose("seal", ls, l.IsVerbose())
	} else {
		event = ll.Debug().
			Dict("seal", logging.Dict().
				Hinted("hint", sl.Hint()).
				Hinted("hash", sl.Hash()).(*logging.Event),
			)
	}

	event.Msg("seal")

	return ll
}

func loggerWithBallot(blt ballot.Ballot, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("ballot_hash", blt.Hash()).(logging.Context).
			CallerWithSkipFrameCount(3)
	})

	ll.Debug().HintedVerbose("ballot", blt, l.IsVerbose()).Msg("ballot")

	return ll
}

func loggerWithVoteproof(voteproof base.Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof", voteproof).(logging.Context).
			CallerWithSkipFrameCount(3)
	})

	ll.Debug().HintedVerbose("voteproof", voteproof, true).Msg("voteproof")

	return ll
}

func loggerWithLocalstate(localstate *Localstate, l logging.Logger) logging.Logger {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return l
	}

	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Dict("local_state", logging.Dict().
			Hinted("block", lastBlock),
		)
	})
}

func loggerWithStateChangeContext(sctx StateChangeContext, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("change_state_context", sctx).(logging.Context).
			CallerWithSkipFrameCount(4)
	})

	return loggerWithVoteproof(sctx.voteproof, ll)
}
