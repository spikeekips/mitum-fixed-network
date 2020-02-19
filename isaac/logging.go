package isaac

import (
	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"

	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

func loggerWithSeal(sl seal.Seal, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().Str("seal_hash", sl.Hash().String()).CallerWithSkipFrameCount(3).Logger()

	if ls, ok := sl.(zerolog.LogObjectMarshaler); ok {
		ll.Debug().EmbedObject(ls).Send()
	} else {
		ll.Debug().
			Dict("seal", zerolog.Dict().
				Str("hint", sl.Hint().Verbose()).
				Str("hash", sl.Hash().String()),
			).Send()
	}

	return &ll
}

func loggerWithBallot(ballot Ballot, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().Str("seal_hash", ballot.Hash().String()).CallerWithSkipFrameCount(3).Logger()

	if lb, ok := ballot.(zerolog.LogObjectMarshaler); ok {
		ll.Debug().EmbedObject(lb).Send()
	} else {
		ll.Debug().
			Dict("ballot", zerolog.Dict().
				Int64("height", ballot.Height().Int64()).
				Uint64("round", ballot.Round().Uint64()).
				Str("stage", ballot.Stage().String()).
				Str("node", ballot.Node().String()),
			).Send()
	}

	return &ll
}

func loggerWithVoteProof(vp VoteProof, l *zerolog.Logger) *zerolog.Logger {
	if vp == nil {
		return l
	}

	ll := l.With().Str("voteproof_id", uuid.Must(uuid.NewV4(), nil).String()).CallerWithSkipFrameCount(3).Logger()

	if lvp, ok := vp.(zerolog.LogObjectMarshaler); ok {
		ll.Debug().EmbedObject(lvp).Send()
	} else if l.GetLevel() == zerolog.DebugLevel {
		rvp, _ := util.JSONMarshal(vp)
		ll.Debug().RawJSON("voteproof", rvp).Send()
	}

	return &ll
}

func loggerWithLocalState(localState *LocalState, l *zerolog.Logger) *zerolog.Logger {
	lastBlock := localState.LastBlock()
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
		).Send()

	return l
}

func loggerWithConsensusStateChangeContext(ctx ConsensusStateChangeContext, l *zerolog.Logger) *zerolog.Logger {
	e := zerolog.Dict().
		Str("from_state", ctx.From().String()).
		Str("to_state", ctx.To().String())

	if ctx.voteProof != nil {
		if lvp, ok := ctx.voteProof.(zerolog.LogObjectMarshaler); ok {
			e.EmbedObject(lvp)
		} else {
			rvp, _ := util.JSONMarshal(ctx.voteProof)

			e.RawJSON("voteproof", rvp)
		}
	}

	ll := l.With().
		Str("change_state_context_id", uuid.Must(uuid.NewV4(), nil).String()).
		CallerWithSkipFrameCount(3).
		Logger()
	ll.Debug().Dict("change_state_context", e).Send()

	return loggerWithVoteProof(ctx.voteProof, &ll)
}
