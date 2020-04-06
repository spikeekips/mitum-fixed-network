package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

func loggerWithSeal(sl seal.Seal, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("seal_hash", sl.Hash().String()).
			CallerWithSkipFrameCount(3)
	})

	var event *zerolog.Event
	if ls, ok := sl.(zerolog.LogObjectMarshaler); ok {
		event = ll.Debug().EmbedObject(ls)
	} else {
		event = ll.Debug().
			Dict("seal", zerolog.Dict().
				Str("hint", sl.Hint().Verbose()).
				Str("hash", sl.Hash().String()),
			)
	}

	event.Msg("seal")

	return ll
}

func loggerWithBallot(ballot Ballot, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("seal_hash", ballot.Hash().String()).
			CallerWithSkipFrameCount(3)
	})

	var event *zerolog.Event
	if lb, ok := ballot.(zerolog.LogObjectMarshaler); ok {
		event = ll.Debug().EmbedObject(lb)
	} else {
		event = ll.Debug().
			Dict("ballot", zerolog.Dict().
				Int64("height", ballot.Height().Int64()).
				Uint64("round", ballot.Round().Uint64()).
				Str("stage", ballot.Stage().String()).
				Str("node", ballot.Node().String()),
			)
	}

	event.Msg("ballot")

	return ll
}

func loggerWithVoteproof(voteproof Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	ll := l.WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("voteproof_id", util.UUID().String()).CallerWithSkipFrameCount(3)
	})

	var event *zerolog.Event
	if lvp, ok := voteproof.(zerolog.LogObjectMarshaler); ok {
		event = ll.Debug().EmbedObject(lvp)
	} else if l.GetLevel() == zerolog.DebugLevel {
		rvp, _ := util.JSONMarshal(voteproof)
		event = ll.Debug().RawJSON("voteproof", rvp)
	}

	event.Msg("voteproof")

	return ll
}

func loggerWithLocalstate(localstate *Localstate, l logging.Logger) logging.Logger {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return l
	}

	l.Debug().
		Dict("local_state", zerolog.Dict().
			Dict("block", zerolog.Dict().
				Str("hash", lastBlock.Hash().String()).
				Int64("height", lastBlock.Height().Int64()).
				Uint64("round", lastBlock.Round().Uint64()),
			),
		).Msg("localstate")

	return l
}

func loggerWithStateChangeContext(ctx StateChangeContext, l logging.Logger) logging.Logger {
	e := zerolog.Dict().
		Str("from_state", ctx.From().String()).
		Str("to_state", ctx.To().String())

	if ctx.voteproof != nil {
		if lvp, ok := ctx.voteproof.(zerolog.LogObjectMarshaler); ok {
			e.EmbedObject(lvp)
		} else {
			rvp, _ := util.JSONMarshal(ctx.voteproof)

			e.RawJSON("voteproof", rvp)
		}
	}

	ll := l.WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("change_state_context_id", util.UUID().String()).
			CallerWithSkipFrameCount(4)
	})

	ll.Debug().Dict("change_state_context", e).Msg("state_change_context")

	return loggerWithVoteproof(ctx.voteproof, ll)
}
