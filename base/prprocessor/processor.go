package prprocessor

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
)

type State int

const (
	BeforePrepared State = iota
	Preparing
	PrepareFailed
	Prepared
	Saving
	SaveFailed
	Saved
	Canceled
)

func (o State) String() string {
	switch o {
	case BeforePrepared:
		return "BeforePrepared"
	case Preparing:
		return "Preparing"
	case Prepared:
		return "Prepared"
	case PrepareFailed:
		return "PrepareFailed"
	case Saving:
		return "Saving"
	case Saved:
		return "Saved"
	case SaveFailed:
		return "SaveFailed"
	case Canceled:
		return "Canceled"
	default:
		return "<unknown processor state>"
	}
}

type Processor interface {
	State() State
	Fact() base.ProposalFact
	SignedFact() base.SignedBallotFact
	SetACCEPTVoteproof(base.Voteproof /* ACCEPT Voteproof */) error
	Prepare(context.Context) (block.Block, error)
	Save(context.Context) error
	Cancel() error
	Block() block.Block
	Statics() map[string]interface{}
}
