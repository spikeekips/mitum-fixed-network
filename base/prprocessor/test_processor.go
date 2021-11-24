package prprocessor

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
)

func (pps *Processors) SetCurrent(pp Processor) {
	pps.setCurrent(pp)
}

type DummyProcessor struct {
	sync.RWMutex
	stateLock sync.RWMutex
	uid       string
	S         State
	P         base.SignedBallotFact
	B         block.Block
	IV        base.Voteproof
	AV        base.Voteproof
	PF        func(context.Context) (block.Block, error)
	SF        func(context.Context) error
}

func (pp *DummyProcessor) New(sfs base.SignedBallotFact, initVoteproof base.Voteproof) (Processor, error) {
	return &DummyProcessor{
		uid: util.UUID().String(),
		S:   pp.S,
		P:   sfs,
		B:   pp.B,
		IV:  initVoteproof,
		AV:  pp.AV,
		PF:  pp.PF,
		SF:  pp.SF,
	}, nil
}

func (pp *DummyProcessor) State() State {
	pp.stateLock.RLock()
	defer pp.stateLock.RUnlock()

	if pp.S < BeforePrepared {
		return BeforePrepared
	}

	return pp.S
}

func (pp *DummyProcessor) SetState(s State) {
	pp.stateLock.Lock()
	defer pp.stateLock.Unlock()

	if pp.S >= Canceled {
		return
	}

	pp.S = s
}

func (pp *DummyProcessor) Fact() base.ProposalFact {
	return pp.P.Fact().(base.ProposalFact)
}

func (pp *DummyProcessor) SignedFact() base.SignedBallotFact {
	return pp.P
}

func (pp *DummyProcessor) SetACCEPTVoteproof(voteproof base.Voteproof) error {
	pp.Lock()
	defer pp.Unlock()

	pp.AV = voteproof

	return nil
}

func (pp *DummyProcessor) Prepare(ctx context.Context) (block.Block, error) {
	pp.SetState(Preparing)

	if pp.PF == nil {
		pp.SetState(PrepareFailed)

		return nil, errors.Errorf("empty Prepare func")
	}

	nctx := context.WithValue(ctx, "proposal", pp.Fact().Hash()) //lint:ignore SA1029 test
	if blk, err := pp.PF(nctx); err != nil {
		pp.SetState(PrepareFailed)

		return nil, err
	} else {
		pp.SetState(Prepared)
		pp.setBlock(blk)

		return blk, nil
	}
}

func (pp *DummyProcessor) Save(ctx context.Context) error {
	pp.SetState(Saving)

	if pp.SF == nil {
		pp.SetState(SaveFailed)

		return util.StopRetryingError.Errorf("empty save func")
	}

	if err := pp.SF(ctx); err != nil {
		pp.SetState(SaveFailed)

		return err
	} else {
		pp.SetState(Saved)

		return nil
	}
}

func (pp *DummyProcessor) Cancel() error {
	pp.SetState(Canceled)

	return nil
}

func (pp *DummyProcessor) Block() block.Block {
	pp.RLock()
	defer pp.RUnlock()

	return pp.B
}

func (pp *DummyProcessor) setBlock(blk block.Block) {
	pp.Lock()
	defer pp.Unlock()

	pp.B = blk
}

func (pp *DummyProcessor) Statics() map[string]interface{} {
	return nil
}
