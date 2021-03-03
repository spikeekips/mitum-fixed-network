package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

func LoggerWithSeal(sl seal.Seal, l logging.Logger) logging.Logger {
	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Interface("seal_hint", sl.Hint()).Hinted("seal_hash", sl.Hash()).(logging.Context)
	})
}

func LoggerWithBallot(blt ballot.Ballot, l logging.Logger) logging.Logger {
	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("ballot_hash", blt.Hash()).(logging.Context)
	})
}

func LoggerWithVoteproof(voteproof base.Voteproof, l logging.Logger) logging.Logger {
	if voteproof == nil {
		return l
	}

	return l.WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("voteproof_id", voteproof.ID()).(logging.Context)
	})
}

func LoggerWithLocal(local *Local, l logging.Logger) logging.Logger {
	var manifest block.Manifest
	if m, found, err := local.Storage().LastManifest(); err != nil || !found {
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
