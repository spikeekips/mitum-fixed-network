package isaac

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

func insideLogContext(ctx logging.Context) logging.Context {
	if ctx.Logger().GetLevel() <= zerolog.DebugLevel {
		ctx = ctx.CallerWithSkipFrameCount(3).(logging.Context)
	}

	return ctx
}

func outsideLogContext(ctx logging.Context) logging.Context {
	if ctx.Logger().GetLevel() <= zerolog.DebugLevel {
		ctx = ctx.CallerWithSkipFrameCount(2).(logging.Context)
	}

	return ctx
}

func outsideLogger(l logging.Logger) logging.Logger {
	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return outsideLogContext(ctx)
	})
}

func loggerWithSeal(sl seal.Seal, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return insideLogContext(ctx).Hinted("seal_hash", sl.Hash()).(logging.Context)
	})

	seal.LoggerWithSeal(sl, ll.Debug(), ll.IsVerbose()).Msg("seal")

	return outsideLogger(ll)
}

func loggerWithBallot(blt ballot.Ballot, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return insideLogContext(ctx).Hinted("ballot_hash", blt.Hash()).(logging.Context)
	})

	ll.Debug().HintedVerbose("ballot", blt, l.IsVerbose()).Msg("ballot")

	return outsideLogger(ll)
}

func loggerWithVoteproofID(voteproof base.Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return outsideLogContext(ctx).Str("voteproof_id", voteproof.ID()).(logging.Context)
	})
}

func loggerWithVoteproof(voteproof base.Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return insideLogContext(ctx).Str("voteproof_id", voteproof.ID()).(logging.Context)
	})

	ll.Info().HintedVerbose("voteproof", voteproof, true).Msg("voteproof")

	return outsideLogger(ll)
}

func loggerWithLocal(local *Local, l logging.Logger) logging.Logger {
	var manifest block.Manifest
	if m, found, err := local.Storage().LastManifest(); err != nil || !found {
		return l
	} else {
		manifest = m
	}

	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return outsideLogContext(ctx).Dict("local_state", logging.Dict().
			Hinted("block", manifest),
		)
	})
}
