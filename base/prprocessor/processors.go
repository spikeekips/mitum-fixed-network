package prprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	PrepareFailedError = util.NewError("failed to prepare")
	SaveFailedError    = util.NewError("failed to save")
)

type Result struct {
	Block block.Block
	Err   error
}

func (r Result) IsEmpty() bool {
	return r.Block == nil && r.Err == nil
}

type ProcessorNewFunc func(base.SignedBallotFact, base.Voteproof) (Processor, error)

type pv struct {
	ctx       context.Context
	sfs       base.SignedBallotFact
	voteproof base.Voteproof
	outchan   chan Result
}

type sv struct {
	ctx       context.Context
	fact      valuehash.Hash
	voteproof base.Voteproof
	outchan   chan Result
}

type Processors struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	newFunc           ProcessorNewFunc
	proposalChecker   func(base.ProposalFact) error
	newProposalChan   chan pv
	saveChan          chan sv
	current           Processor
	cancelPrepareFunc func()
	cancelSaveFunc    func()
}

func NewProcessors(newFunc ProcessorNewFunc, proposalChecker func(base.ProposalFact) error) *Processors {
	pps := &Processors{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "default-proposal-processors")
		}),
		newFunc:         newFunc,
		proposalChecker: proposalChecker,
		newProposalChan: make(chan pv),
		saveChan:        make(chan sv),
	}

	pps.ContextDaemon = util.NewContextDaemon("default-proposal-processors", pps.start)

	return pps
}

func (*Processors) Initialize() error {
	return nil
}

func (pps *Processors) NewProposal(
	ctx context.Context,
	sfs base.SignedBallotFact,
	initVoteproof base.Voteproof,
) <-chan Result {
	// NOTE 1-size bufferred channel; channel can be closed without if receiver
	// does not receive from channel
	if initVoteproof.Stage() != base.StageINIT {
		ch := make(chan Result)

		go func() {
			ch <- Result{Err: errors.Errorf("not valid voteproof, %v", initVoteproof.Stage())}

			close(ch)
		}()

		return ch
	}

	ch := make(chan Result, 1)

	go func() {
		pps.newProposalChan <- pv{ctx: ctx, sfs: sfs, voteproof: initVoteproof, outchan: ch}
	}()

	return ch
}

func (pps *Processors) Save(
	ctx context.Context,
	fact valuehash.Hash,
	acceptVoteproof base.Voteproof,
) <-chan Result {
	if acceptVoteproof.Stage() != base.StageACCEPT {
		ch := make(chan Result)

		go func() {
			ch <- Result{Err: errors.Errorf("not valid voteproof, %v", acceptVoteproof.Stage())}

			close(ch)
		}()

		return ch
	}

	ch := make(chan Result, 1)

	go func() {
		pps.saveChan <- sv{ctx: ctx, fact: fact, voteproof: acceptVoteproof, outchan: ch}
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

func (pps *Processors) start(ctx context.Context) error {
end:
	for {
		select {
		case <-ctx.Done():
			break end
		case i := <-pps.newProposalChan:
			r := pps.handleProposal(i.ctx, i.sfs, i.voteproof, i.outchan) // nolint:contextcheck
			if err := r.Err; err != nil {
				l := pps.Log().With().Stringer("proposal", i.sfs.Fact().Hash()).Err(err).Logger()

				if errors.Is(err, util.IgnoreError) {
					l.Debug().Msg("proposal ignored")
				} else {
					l.Error().Msg("failed to handle proposal")
				}
			}

			if !r.IsEmpty() {
				go func(ch chan<- Result) {
					ch <- r
				}(i.outchan)
			}
		case i := <-pps.saveChan:
			if r := pps.saveProposal(i.ctx, i.fact, i.voteproof, i.outchan); !r.IsEmpty() { // nolint:contextcheck
				go func(ch chan<- Result) {
					ch <- r
				}(i.outchan)
			} else if err := r.Err; err != nil {
				if errors.Is(err, util.IgnoreError) {
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
	sfs base.SignedBallotFact,
	initVoteproof base.Voteproof,
	outchan chan<- Result,
) Result {
	fact := sfs.Fact().(base.ProposalFact)
	if err := pps.checkProposal(fact); err != nil {
		return Result{Err: PrepareFailedError.Merge(err)}
	}

	var current Processor
	switch pp, err := pps.checkCurrent(fact); {
	case err != nil:
		return Result{Err: PrepareFailedError.Merge(err)}
	default:
		if pp == nil {
			p, err := pps.newProcessor(sfs, initVoteproof)
			if err != nil {
				return Result{Err: err}
			}

			pps.setCurrent(p)

			pp = p
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
	l := pps.Log().With().
		Int64("height", processor.Fact().Height().Int64()).
		Uint64("round", processor.Fact().Round().Uint64()).
		Stringer("proposal", processor.Fact().Hash()).
		Logger()

	var blk block.Block
	err := util.Retry(3, time.Millisecond*200, func(int) error {
		select {
		case <-ctx.Done():
			l.Error().Err(ctx.Err()).Msg("something wrong to prepare; will be stopped")

			return util.StopRetryingError.Wrap(ctx.Err())
		default:
			switch b, err := processor.Prepare(ctx); {
			case err == nil:
				blk = b

				return nil
			case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
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
		err = PrepareFailedError.Merge(err)

		l.Error().Err(err).Msg("failed to prepare; processor will be canceled")

		switch processor.State() {
		case Prepared, BeforePrepared, PrepareFailed, SaveFailed, Saved, Canceled:
		default:
			if cerr := pps.cancelProcessor(processor); cerr != nil {
				l.Error().Err(err).Msg("failed to cancel processor")
			}
		}
	} else if blk != nil {
		l.Debug().Stringer("new_block", blk.Hash()).Msg("new block prepared")
	}

	if err != nil {
		pps.setCurrent(nil)
	}

	outchan <- Result{Block: blk, Err: err}
}

func (pps *Processors) saveProposal(
	ctx context.Context,
	fact valuehash.Hash,
	acceptVoteproof base.Voteproof,
	outchan chan<- Result,
) Result {
	current := pps.Current()

	var err error
	if current == nil {
		err = errors.Errorf("not yet prepared")
	} else if h := current.Fact().Hash(); !h.Equal(fact) { // NOTE if different processor exists already
		err = errors.Errorf("not yet prepared; another processor already exists")

		LogEventProcessor(current, "current", pps.Log().Error().Err(err)).
			Stringer("propsoal", fact).
			Msg("failed to save proposal")
	}

	if err != nil {
		return Result{Err: SaveFailedError.Merge(err)}
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
	l := pps.Log().With().
		Int64("height", processor.Fact().Height().Int64()).
		Uint64("round", processor.Fact().Round().Uint64()).
		Stringer("proposal", processor.Fact().Hash()).
		Logger()

	// NOTE tries 3 times
	err := util.Retry(3, time.Millisecond*200, func(int) error {
		select {
		case <-ctx.Done():
			l.Error().Err(ctx.Err()).Msg("something wrong to save; will be stopped")

			return util.StopRetryingError.Wrap(ctx.Err())
		default:
			switch err := pps.save(ctx, processor, acceptVoteproof); {
			case err == nil:
				return nil
			case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
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

	var blk block.Block
	if err == nil {
		blk = processor.Block()
	} else {
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

	outchan <- Result{Block: blk, Err: err}
}

func (pps *Processors) save(ctx context.Context, processor Processor, acceptVoteproof base.Voteproof) error {
	switch processor.State() {
	case BeforePrepared:
		return errors.Errorf("not yet prepared")
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
		return errors.Errorf("failed to prepare")
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

func (pps *Processors) CurrentState(fact valuehash.Hash) State {
	switch current := pps.Current(); {
	case current == nil:
		return BeforePrepared
	case !current.Fact().Hash().Equal(fact):
		return BeforePrepared
	default:
		return current.State()
	}
}

func (pps *Processors) checkCurrent(fact base.ProposalFact) (Processor, error) {
	current := pps.Current()

	if current == nil {
		return nil, nil
	}

	if h := current.Fact().Hash(); !h.Equal(fact.Hash()) {
		cpr := current.Fact()

		if cpr.Height() == fact.Height() && cpr.Round() == fact.Round() {
			return nil, util.IgnoreError.Errorf("duplicated proposal received")
		}

		if current.State() != Saved {
			LogEventProcessor(current, "current", pps.Log().Debug()).
				Stringer("proposal", fact.Hash()).Bool("current_exists", current == nil).
				Msg("found previous Processor with different Proposal; existing Processor will be canceled")

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
		return nil, errors.Errorf("unknow current state, %s", state)
	}
}

func (pps *Processors) newProcessor(sfs base.SignedBallotFact, initVoteproof base.Voteproof) (Processor, error) {
	if pp, err := pps.newFunc(sfs, initVoteproof); err != nil {
		return nil, PrepareFailedError.Wrap(err)
	} else if state := pp.State(); state != BeforePrepared {
		return nil, PrepareFailedError.Errorf("new Processor should be BeforePrepared state, not %s", state)
	} else {
		if l, ok := pp.(logging.SetLogging); ok {
			_ = l.SetLogging(pps.Logging)
		}

		return pp, nil
	}
}

func (pps *Processors) checkProposal(fact base.ProposalFact) error {
	if fact == nil || fact.Hash() == nil {
		return errors.Errorf("invalid proposal")
	}

	if pps.proposalChecker != nil {
		if err := pps.proposalChecker(fact); err != nil {
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
