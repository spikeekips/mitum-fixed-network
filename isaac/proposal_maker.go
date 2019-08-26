package isaac

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type ProposalMaker interface {
	Make(Height, Round, hash.Hash /* last block */) (Proposal, error)
}

type DefaultProposalMaker struct {
	*common.Logger
	home  node.Home
	delay time.Duration
}

func NewDefaultProposalMaker(home node.Home, delay time.Duration) DefaultProposalMaker {
	return DefaultProposalMaker{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "proposer-maker")
		}),
		home:  home,
		delay: delay,
	}
}

func (dp DefaultProposalMaker) Make(height Height, round Round, lastBlock hash.Hash) (Proposal, error) {
	log_ := dp.Log().With().
		Uint64("height", height.Uint64()).
		Uint64("round", round.Uint64()).
		Object("last_block", lastBlock).
		Dur("delay", dp.delay).
		Logger()
	log_.Debug().Msg("ready to make new proposal")

	var started time.Time
	if dp.delay > 0 {
		started = time.Now()
	}

	proposal, err := NewProposal(
		height,
		round,
		lastBlock,
		dp.home.Address(),
		nil, // TODO transactions
	)
	if err != nil {
		return Proposal{}, err
	}

	if dp.delay > 0 {
		since := time.Since(started)
		if dp.delay > since {
			time.Sleep(dp.delay - since)
		}
	}

	log_.Debug().Msg("new proposal created")

	return proposal, nil
}
