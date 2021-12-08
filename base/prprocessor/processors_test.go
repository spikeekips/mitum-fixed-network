package prprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProcessors struct {
	suite.Suite
	pk key.Privatekey
}

func (t *testProcessors) SetupSuite() {
	t.pk = key.NewBasePrivatekey()
}

func (t *testProcessors) newProposal(height base.Height, round base.Round) base.SignedBallotFact {
	n := base.RandomStringAddress()
	fact := ballot.NewProposalFact(
		height,
		round,
		n,
		nil,
	)

	sfs, _ := base.NewBaseSignedBallotFactFromFact(fact, n, t.pk, nil)

	return sfs
}

func (t *testProcessors) newVoteproof(height base.Height, round base.Round, stage base.Stage) base.Voteproof {
	vp := base.NewTestVoteproofV0(
		height,
		round,
		nil,
		base.ThresholdRatio(67),
		base.VoteResultMajority,
		false,
		stage,
		nil,
		nil,
		nil,
		localtime.UTCNow(),
	)

	return vp
}

func (t *testProcessors) TestNew() {
	pp := &DummyProcessor{}
	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	t.Nil(pps.Current())
}

func (t *testProcessors) TestNewProposal() {
	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		// returns error with nil block
		return nil, util.StopRetryingError.Errorf("showme")
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.Nil(result.Block)
		t.Contains(result.Err.Error(), "showme")
	}
}

func (t *testProcessors) TestNewProposalDuplicatedProposal() {
	height, round := base.Height(33), base.Round(33)
	pr0 := t.newProposal(height, round)

	newblock, err := block.NewTestBlockV0(height, round, pr0.Fact().Hash(), valuehash.RandomSHA256())
	t.NoError(err)

	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	ivp := t.newVoteproof(height, round, base.StageINIT)

	ch0 := pps.NewProposal(context.Background(), pr0, ivp)
	result0 := <-ch0
	t.NotNil(result0.Block)
	t.Nil(result0.Err)

	pr1 := t.newProposal(height, round)
	t.False(pr0.Fact().Hash().Equal(pr1.Fact().Hash()))

	ch1 := pps.NewProposal(context.Background(), pr1, ivp)
	result1 := <-ch1
	t.NotNil(result1.Err)
	t.True(errors.Is(result1.Err, util.IgnoreError))
	t.Contains(result1.Err.Error(), "duplicated proposal received")
}

func (t *testProcessors) TestNewProposalWithWrongVoteroof() {
	pp := &DummyProcessor{}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageACCEPT)

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.Nil(result.Block)
		t.Contains(result.Err.Error(), "not valid voteproof")
	}
}

func (t *testProcessors) TestPrepareTimeout() {
	timeout := time.Millisecond * 300
	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(timeout + time.Second*100):
			return nil, nil
		}
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := pps.NewProposal(ctx, pr, ivp)

	select {
	case <-time.After(timeout + time.Second*1):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.Nil(result.Block)
		t.Contains(result.Err.Error(), "context deadline exceeded")
	}
}

func (t *testProcessors) TestCancelPreviousProcessors() {
	canceled := make(chan valuehash.Hash, 100)
	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		<-time.After(time.Second)

		canceled <- ctx.Value("proposal").(valuehash.Hash)

		return nil, nil
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	var previous []Processor
	{
		height, round := base.Height(33), base.Round(33)
		pr := t.newProposal(height, round)
		ivp := t.newVoteproof(height, round, base.StageINIT)

		_ = pps.NewProposal(context.Background(), pr, ivp)
	}

	<-time.After(time.Millisecond * 10)

	t.NotNil(pps.Current())
	t.Equal(Preparing, pps.Current().State())
	previous = append(previous, pps.Current())

	for i := 0; i < 5; i++ { // add one more Processor
		height, round := base.Height(34+i), base.Round(3)
		pr := t.newProposal(height, round)
		ivp := t.newVoteproof(height, round, base.StageINIT)

		_ = pps.NewProposal(context.Background(), pr, ivp)

		for {
			<-time.After(time.Millisecond * 10)
			if pps.Current().Fact().Hash().Equal(pr.Fact().Hash()) {
				previous = append(previous, pps.Current())

				break
			}
		}
	}

	finished := make(chan []valuehash.Hash)
	go func() {
		var hs []valuehash.Hash
		for pr := range canceled {
			hs = append(hs, pr)

			if len(hs) == len(previous) {
				break
			}
		}

		finished <- hs
	}()

	finishedProposals := <-finished

	for i, p := range previous {
		if i == len(previous)-1 {
			t.Equal(Prepared, p.State(), "total=%d i=%d", len(previous), i)
		} else {
			t.Equal(Canceled, p.State(), "total=%d i=%d", len(previous), i)
		}

		t.True(finishedProposals[i].Equal(p.Fact().Hash()))
	}
}

func (t *testProcessors) TestPrepareExistsAsSaved() {
	height, round := base.Height(33), base.Round(33)
	pr := t.newProposal(height, round)

	newblock, err := block.NewTestBlockV0(height, round, pr.Fact().Hash(), valuehash.RandomSHA256())
	t.NoError(err)

	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	var ivp base.Voteproof
	{
		ivp = t.newVoteproof(height, round, base.StageINIT)

		_ = pps.NewProposal(context.Background(), pr, ivp)

		<-time.After(time.Millisecond * 50)
		t.Equal(Prepared, pps.Current().State())

		// reset to PrepareFailed
		pps.Current().(*DummyProcessor).SetState(Saved)
	}

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.Nil(result.Block)
		t.Equal(Saved, pps.Current().State())
	}
}

func (t *testProcessors) TestPrepareExistsAsFailed() {
	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		return block.BlockV0{}, nil
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	var pr base.SignedBallotFact
	var ivp base.Voteproof
	{
		height, round := base.Height(33), base.Round(33)
		pr = t.newProposal(height, round)
		ivp = t.newVoteproof(height, round, base.StageINIT)

		_ = pps.NewProposal(context.Background(), pr, ivp)

		<-time.After(time.Millisecond * 50)
		t.Equal(Prepared, pps.Current().State())

		// reset to PrepareFailed
		pps.Current().(*DummyProcessor).SetState(PrepareFailed)
	}

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.NotNil(result.Block)
		t.IsType(block.BlockV0{}, result.Block)
		t.Equal(Prepared, pps.Current().State())
	}
}

func (t *testProcessors) TestSaveButNotYetPrepared() {
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			return nil, nil
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)
	pr := t.newProposal(height, round)
	avp := t.newVoteproof(height, round, base.StageACCEPT)

	// save
	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.True(errors.Is(result.Err, SaveFailedError))
		t.Contains(result.Err.Error(), "not yet prepared")
	}
}

func (t *testProcessors) TestSaveButPrepareFailed() {
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			return nil, nil
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)
	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)
	avp := t.newVoteproof(height, round, base.StageACCEPT)

	pch := pps.NewProposal(context.Background(), pr, ivp)
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.Nil(result.Block)
		t.NoError(result.Err)
		t.Equal(Prepared, pps.Current().State())

		// reset to PrepareFailed
		pps.Current().(*DummyProcessor).SetState(PrepareFailed)
	}

	// save
	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(PrepareFailed, pps.Current().State())
		t.Contains(result.Err.Error(), "failed to prepare")
	}
}

func (t *testProcessors) TestEmptySaveFunc() {
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			return nil, nil
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)
	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)
	avp := t.newVoteproof(height, round, base.StageACCEPT)

	pch := pps.NewProposal(context.Background(), pr, ivp)
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.Nil(result.Block)
		t.NoError(result.Err)
		t.Equal(Prepared, pps.Current().State())
	}

	// save
	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(SaveFailed, pps.Current().State())
		t.Contains(result.Err.Error(), "empty save func")
	}
}

func (t *testProcessors) TestSaveTimeout() {
	timeout := time.Millisecond * 300
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			return block.BlockV0{}, nil
		},
		SF: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(timeout + time.Second*100):
				return nil
			}
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	pch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(Prepared, pps.Current().State())
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	avp := t.newVoteproof(height, round, base.StageACCEPT)

	sch := pps.Save(ctx, pr.Fact().Hash(), avp)

	select {
	case <-time.After(timeout + time.Second*1):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Nil(result.Block)
		t.Contains(result.Err.Error(), "context deadline exceeded")
	}
}

func (t *testProcessors) TestSaveWaitPrepapred() {
	timeout := time.Second * 1
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			<-time.After(timeout)

			return block.BlockV0{}, nil
		},
		SF: func(ctx context.Context) error {
			return nil
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	_ = pps.NewProposal(context.Background(), pr, ivp)

end:
	for {
		select {
		case <-time.After(time.Second * 3):
			t.NoError(errors.Errorf("waiting result, but expired"))

			return
		default:
			if pps.Current() == nil || +pps.Current().State() != Preparing {
				<-time.After(time.Millisecond * 10)
			}

			break end
		}
	}

	avp := t.newVoteproof(height, round, base.StageACCEPT)

	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.NotNil(result.Block)
		t.NoError(result.Err)
	}
}

func (t *testProcessors) TestProposalChecker() {
	pps := NewProcessors(nil, func(base.ProposalFact) error {
		return errors.Errorf("checker pong pong")
	})
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.Nil(result.Block)
		t.Contains(result.Err.Error(), "checker pong pong")
	}
}

func (t *testProcessors) TestPrepareRetry() {
	var i int
	pp := &DummyProcessor{PF: func(ctx context.Context) (block.Block, error) {
		if i < 1 {
			i++
			return nil, errors.Errorf("showme")
		}

		return block.BlockV0{}, nil
	}}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	ch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-ch:
		t.NotNil(result.Block)
		t.NoError(result.Err)
	}
}

func (t *testProcessors) TestSaveRetry() {
	var i int
	pp := &DummyProcessor{
		PF: func(ctx context.Context) (block.Block, error) {
			return block.BlockV0{}, nil
		},
		SF: func(ctx context.Context) error {
			if i < 1 {
				i++
				return errors.Errorf("showme")
			}

			return nil
		},
	}

	pps := NewProcessors(pp.New, nil)
	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	height, round := base.Height(33), base.Round(33)

	pr := t.newProposal(height, round)
	ivp := t.newVoteproof(height, round, base.StageINIT)

	pch := pps.NewProposal(context.Background(), pr, ivp)
	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("waiting result, but expired to prepare"))

		return
	case result := <-pch:
		t.NoError(result.Err)
	}

	avp := t.newVoteproof(height, round, base.StageACCEPT)

	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("waiting result, but expired to save"))

		return
	case result := <-sch:
		t.NoError(result.Err)
	}
}

func TestProcessors(t *testing.T) {
	suite.Run(t, new(testProcessors))
}
