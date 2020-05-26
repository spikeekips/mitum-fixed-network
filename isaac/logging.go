package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

func loggerWithSeal(sl seal.Seal, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("seal_hash", sl.Hash()).(logging.Context)
	})

	seal.LoggerWithSeal(sl, ll.Debug(), ll.IsVerbose()).Msg("seal")

	return ll
}

func loggerWithBallot(blt ballot.Ballot, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("ballot_hash", blt.Hash()).(logging.Context)
	})

	ll.Debug().HintedVerbose("ballot", blt, l.IsVerbose()).Msg("ballot")

	return ll
}

func loggerWithVoteproof(voteproof base.Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof", voteproof).(logging.Context)
	})

	ll.Debug().HintedVerbose("voteproof", voteproof, true).Msg("voteproof")

	return ll
}

func loggerWithLocalstate(localstate *Localstate, l logging.Logger) logging.Logger {
	var manifest block.Manifest
	if m, err := localstate.Storage().LastManifest(); err != nil {
		return l
	} else {
		manifest = m
	}

	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Dict("local_state", logging.Dict().
			Hinted("block", manifest),
		)
	})
}

func loggerWithStateChangeContext(sctx StateChangeContext, l logging.Logger) logging.Logger {
	ll := l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("change_state_context", sctx).(logging.Context)
	})

	return loggerWithVoteproof(sctx.voteproof, ll)
}
