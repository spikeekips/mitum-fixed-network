package prprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var (
	PrepareFailedError = errors.NewError("failed to prepare")
	SaveFailedError    = errors.NewError("failed to save")
)

type Result struct {
	Block block.Block
	Err   error
}

func (r Result) IsEmpty() bool {
	return r.Block == nil && r.Err == nil
}

type ProcessorNewFunc func(ballot.Proposal, base.Voteproof) (Processor, error)

type pv struct {
	ctx       context.Context
	proposal  ballot.Proposal
	voteproof base.Voteproof
	outchan   chan Result
}

type sv struct {
	ctx       context.Context
	proposal  valuehash.Hash
	voteproof base.Voteproof
	outchan   chan Result
}

type Processors struct {
	sync.RWMutex
	*logging.Logging
	*util.FunctionDaemon
	newFunc           ProcessorNewFunc
	proposalChecker   func(ballot.Proposal) error
	newProposalChan   chan pv
	saveChan          chan sv
	current           Processor
	cancelPrepareFunc func()
	cancelSaveFunc    func()
}

func NewProcessors(newFunc ProcessorNewFunc, proposalChecker func(ballot.Proposal) error) *Processors {
	pps := &Processors{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "default-proposal-processors")
		}),
		newFunc:         newFunc,
		proposalChecker: proposalChecker,
		newProposalChan: make(chan pv),
		saveChan:        make(chan sv),
	}

	pps.FunctionDaemon = util.NewFunctionDaemon(pps.start, false)

	return pps
}

func (pps *Processors) Initialize() error {
	return nil
}

func (pps *Processors) NewProposal(
	ctx context.Context,
	proposal ballot.Proposal,
	initVoteproof base.Voteproof,
) <-chan Result {
	// NOTE 1-size bufferred channel; channel can be closed without if receiver
	// does not receive from channel
	ch := make(chan Result, 1)
	if initVoteproof.Stage() != base.StageINIT {
		go func() {
			ch <- Result{Err: xerrors.Errorf("not valid voteproof, %v", initVoteproof.Stage())}

			close(ch)
		}()

		return ch
	}

	go func() {
		pps.newProposalChan <- pv{ctx: ctx, proposal: proposal, voteproof: initVoteproof, outchan: ch}
	}()

	return ch
}

func (pps *Processors) Save(
	ctx context.Context,
	proposal valuehash.Hash,
	acceptVoteproof base.Voteproof,
) <-chan Result {
	ch := make(chan Result, 1)

	go func() {
		pps.saveChan <- sv{ctx: ctx, proposal: proposal, voteproof: acceptVoteproof, outchan: ch}
	}()

	return ch
}

func (pps *Processors) Current() Processor {
	pps.RLock()
	defer pps.RUnlock()

	return pps.current
}

func (pps *Processors) setCurrent(pp Processor) {
	pps.Lock()
	defer pps.Unlock()

	pps.current = pp
}

func (pps *Processors) start(stopChan chan struct{}) error {
end:
	for {
		select {
		case <-stopChan:
			break end
		case i := <-pps.newProposalChan:
			if r := pps.handleProposal(i.ctx, i.proposal, i.voteproof, i.outchan); !r.IsEmpty() {
				go func(ch chan<- Result) {
					ch <- r
				}(i.outchan)
			} else if err := r.Err; err != nil {
				if xerrors.Is(err, util.IgnoreError) {
					pps.Log().Debug().Err(err).Msg("proposal ignored")
				} else {
					pps.Log().Error().Err(err).Msg("failed to handle proposal")
				}
			}
		case i := <-pps.saveChan:
			if r := pps.saveProposal(i.ctx, i.proposal, i.voteproof, i.outchan); !r.IsEmpty() {
				go func(ch chan<- Result) {
					ch <- r
				}(i.outchan)
			} else if err := r.Err; err != nil {
				if xerrors.Is(err, util.IgnoreError) {
					pps.Log().Debug().Err(err).Msg("saving proposal ignored")
				} else {
					pps.Log().Error().Err(err).Msg("failed to save proposal")
				}
			}
		}
	}

	return nil
}

func (pps *Processors) handleProposal(
	ctx context.Context,
	proposal ballot.Proposal,
	initVoteproof base.Voteproof,
	outchan chan<- Result,
) Result {
	if err := pps.checkProposal(proposal); err != nil {
		return Result{Err: PrepareFailedError.Wrap(err)}
	}

	var current Processor
	switch pp, err := pps.checkPrepareCurrent(proposal.Hash()); {
	case err != nil:
		return Result{Err: PrepareFailedError.Wrap(err)}
	default:
		if pp == nil {
			if p, err := pps.newProcessor(proposal, initVoteproof); err != nil {
				return Result{Err: err}
			} else {
				pps.setCurrent(p)

				pp = p
			}
		}

		current = pp
	}

	go blockingFinished(ctx, func(ctx context.Context, cancel func()) {
		pps.cancelPrepareFunc = cancel

		pps.doPrepare(ctx, current, outchan)
	})

	return Result{}
}

func (pps *Processors) doPrepare(ctx context.Context, processor Processor, outchan chan<- Result) {
	l := pps.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.
			Hinted("height", processor.Proposal().Height()).
			Hinted("round", processor.Proposal().Round()).
			Hinted("proposal", processor.Proposal().Hash())
	})

	var blk block.Block
	err := util.Retry(0, time.Millisecond*200, func() error {
		select {
		case <-ctx.Done():
			l.Error().Err(ctx.Err()).Msg("something wrong to prepare; will be stopped")

			return util.StopRetryingError.Wrap(ctx.Err())
		default:
			switch b, err := processor.Prepare(ctx); {
			case err == nil:
				blk = b

				return nil
			case xerrors.Is(err, context.DeadlineExceeded) || xerrors.Is(err, context.Canceled):
				return util.StopRetryingError.Wrap(err)
			case processor.State() == Canceled:
				return util.StopRetryingError.Errorf("canceled")
			default:
				l.Error().Err(err).Msg("something wrong to prepare; will retry")

				return err
			}
		}
	})
	if err != nil {
		err = PrepareFailedError.Wrap(err)

		l.Error().Err(err).Msg("failed to prepare; processor will be canceled")

		switch processor.State() {
		case Prepared, BeforePrepared, PrepareFailed, SaveFailed, Saved, Canceled:
		default:
			if cerr := pps.cancelProcessor(processor); cerr != nil {
				l.Error().Err(err).Msg("failed to cancel processor")
			}
		}
	} else if blk != nil {
		l.Debug().Hinted("new_block", blk.Hash()).Msg("new block prepared")
	}

	outchan <- Result{Block: blk, Err: err}
}

func (pps *Processors) saveProposal(
	ctx context.Context,
	proposal valuehash.Hash,
	acceptVoteproof base.Voteproof,
	outchan chan<- Result,
) Result {
	current := pps.Current()

	var err error
	if current == nil {
		err = xerrors.Errorf("not yet prepared")
	} else if h := current.Proposal().Hash(); !h.Equal(proposal) { // NOTE if different processor exists already
		err = xerrors.Errorf("not yet prepared; another processor already exists")

		pps.Log().Error().Err(err).
			Dict("previous", logging.Dict().
				Str("state", current.State().String()).
				Hinted("height", current.Proposal().Height()).
				Hinted("round", current.Proposal().Round()).
				Hinted("proposal", current.Proposal().Hash())).
			Hinted("propsoal", proposal).
			Msg("failed to save proposal")
	}

	if err != nil {
		return Result{Err: SaveFailedError.Wrap(err)}
	}

	go blockingFinished(ctx, func(ctx context.Context, cancel func()) {
		pps.cancelSaveFunc = cancel

		pps.doSave(ctx, current, acceptVoteproof, outchan)
	})

	return Result{}
}

func (pps *Processors) doSave(
	ctx context.Context,
	processor Processor,
	acceptVoteproof base.Voteproof,
	outchan chan<- Result,
) {
	l := pps.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.
			Hinted("height", processor.Proposal().Height()).
			Hinted("round", processor.Proposal().Round()).
			Hinted("proposal", processor.Proposal().Hash())
	})

	err := util.Retry(0, time.Millisecond*200, func() error {
		select {
		case <-ctx.Done():
			l.Error().Err(ctx.Err()).Msg("something wrong to save; will be stopped")

			return util.StopRetryingError.Wrap(ctx.Err())
		default:
			err := pps.save(ctx, processor, acceptVoteproof)
			switch {
			case err == nil:
				return nil
			case xerrors.Is(err, context.DeadlineExceeded) || xerrors.Is(err, context.Canceled):
				return util.StopRetryingError.Wrap(err)
			case processor.State() == Canceled:
				return util.StopRetryingError.Errorf("canceled")
			case processor.State() < Prepared:
				return util.StopRetryingError.Wrap(err)
			default:
				l.Error().Err(err).Msg("something wrong to save; will retry")

				return err
			}
		}
	})
	if err != nil {
		err = SaveFailedError.Wrap(err)

		l.Error().Err(err).Msg("failed to save; processor will be canceled")

		switch processor.State() {
		case Prepared, BeforePrepared, PrepareFailed, SaveFailed, Saved, Canceled:
		default:
			if cerr := pps.cancelProcessor(processor); cerr != nil {
				l.Error().Err(err).Msg("failed to cancel processor")
			}
		}
	}

	outchan <- Result{Err: err}
}

func (pps *Processors) save(ctx context.Context, processor Processor, acceptVoteproof base.Voteproof) error {
	switch processor.State() {
	case BeforePrepared:
		return xerrors.Errorf("not yet prepared")
	case Preparing:
		pps.Log().Debug().Msg("Processor is still preparing; will wait")

	end:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if processor.State() != Prepared {
					<-time.After(time.Millisecond * 300)

					continue
				}

				break end
			}
		}
	case Prepared:
		//
	case PrepareFailed:
		return xerrors.Errorf("failed to prepare")
	case Saving:
		return util.IgnoreError.Errorf("already saving")
	case Saved:
		return util.IgnoreError.Errorf("already saved")
	case SaveFailed:
	case Canceled:
		return util.IgnoreError.Errorf("canceled")
	}

	if err := processor.SetACCEPTVoteproof(acceptVoteproof); err != nil {
		return err
	}

	return processor.Save(ctx)
}

func (pps *Processors) checkPrepareCurrent(proposal valuehash.Hash) (Processor, error) {
	current := pps.Current()

	l := pps.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal", proposal).Bool("current_exists", current == nil)
	})

	if current == nil {
		l.Debug().Msg("empty current Processor")

		return nil, nil
	}

	l = pps.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Dict("previous", logging.Dict().
			Str("state", current.State().String()).
			Hinted("proposal", current.Proposal().Hash()),
		)
	})

	if h := current.Proposal().Hash(); h.Equal(proposal) {
		l.Debug().Msg("existing is same Proposal")
	} else {
		l.Debug().Dict("previous", logging.Dict().
			Str("state", current.State().String()).
			Hinted("proposal", current.Proposal().Hash())).
			Msg("found previous Processor with different Proposal")

		if current.State() != Saved {
			l.Debug().Msg("existing Processor will be canceled")

			if err := pps.cancelProcessor(current); err != nil {
				return nil, PrepareFailedError.Wrap(err)
			}
		}

		return nil, nil
	}

	switch state := current.State(); state {
	case BeforePrepared:
		return current, nil
	case Preparing:
		return nil, util.IgnoreError.Errorf("already preparing")
	case Prepared:
		return nil, util.IgnoreError.Errorf("already prepared")
	case Saving:
		return nil, util.IgnoreError.Errorf("already saving")
	case Saved:
		return nil, util.IgnoreError.Errorf("already saved")
	case PrepareFailed, SaveFailed: // NOTE if failed, restart current
		return nil, nil
	case Canceled:
		return nil, util.IgnoreError.Errorf("already canceled")
	default:
		return nil, xerrors.Errorf("unknow current state, %s", state)
	}
}

func (pps *Processors) newProcessor(proposal ballot.Proposal, initVoteproof base.Voteproof) (Processor, error) {
	if pp, err := pps.newFunc(proposal, initVoteproof); err != nil {
		return nil, PrepareFailedError.Wrap(err)
	} else if state := pp.State(); state != BeforePrepared {
		return nil, PrepareFailedError.Errorf("new Processor should be BeforePrepared state, not %s", state)
	} else {
		if l, ok := pp.(logging.SetLogger); ok {
			_ = l.SetLogger(pps.Log())
		}

		return pp, nil
	}
}

func (pps *Processors) checkProposal(proposal ballot.Proposal) error {
	if proposal == nil || proposal.Hash() == nil {
		return xerrors.Errorf("invalid proposal")
	}

	if pps.proposalChecker != nil {
		if err := pps.proposalChecker(proposal); err != nil {
			return err
		}
	}

	return nil
}

func (pps *Processors) cancelProcessor(processor Processor) error {
	if pps.cancelPrepareFunc != nil {
		pps.cancelPrepareFunc()
	}

	if pps.cancelSaveFunc != nil {
		pps.cancelSaveFunc()
	}

	return processor.Cancel()
}

func blockingFinished(ctx context.Context, f func(context.Context, func())) {
	finished := make(chan struct{})

	nctx, cancel := context.WithCancel(ctx)
	go func() {
		f(nctx, cancel)

		finished <- struct{}{}
	}()

	<-finished
}
