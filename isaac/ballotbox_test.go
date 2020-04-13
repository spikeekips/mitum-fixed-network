package isaac

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testBallotbox struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testBallotbox) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotbox) thresholdFunc(total uint, percent float64) func() base.Threshold {
	ls, err := NewLocalstate(nil, nil, TestNetworkID)
	t.NoError(err)

	threshold, _ := base.NewThreshold(total, percent)
	_ = ls.Policy().SetThreshold(threshold)

	return func() base.Threshold {
		return threshold
	}
}

func (t *testBallotbox) TestNew() {
	bb := NewBallotbox(t.thresholdFunc(2, 67))
	ba := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("test-for-init-ballot"), nil, 0)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.NotEmpty(vp)
}

func (t *testBallotbox) newINITBallot(
	height base.Height,
	round base.Round,
	node base.Address,
	previousBlock valuehash.Hash,
	previousRound base.Round,
) ballot.INITBallotV0 {
	vp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	if previousBlock == nil {
		previousBlock = valuehash.RandomSHA256()
	}

	ib := ballot.NewINITBallotV0(
		node,
		height,
		round,
		previousBlock,
		previousRound,
		vp,
	)
	t.NoError(ib.Sign(t.pk, nil))

	return ib
}

func (t *testBallotbox) TestVoteRace() {
	bb := NewBallotbox(t.thresholdFunc(50, 100))

	checkDone := make(chan bool)
	vrChan := make(chan interface{}, 49)

	go func() {
		for i := range vrChan {
			switch c := i.(type) {
			case error:
				t.NoError(c)
			case base.Voteproof:
				t.Equal(base.VoteResultNotYet, c.Result())
			}
		}
		checkDone <- true
	}()

	var wg sync.WaitGroup
	wg.Add(49)
	for i := 0; i < 49; i++ {
		go func() {
			defer wg.Done()
			ba := t.newINITBallot(base.Height(10), base.Round(0), base.RandomShortAddress(), nil, 0)

			vp, err := bb.Vote(ba)
			if err != nil {
				vrChan <- err
			} else {
				vrChan <- vp
			}
		}()
	}
	wg.Wait()
	close(vrChan)

	<-checkDone
}

func (t *testBallotbox) TestINITVoteResultNotYet() {
	bb := NewBallotbox(t.thresholdFunc(2, 67))
	ba := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("test-for-init-ballot"), nil, 0)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(base.VoteResultNotYet, vp.Result())

	t.Equal(ba.Height(), vp.Height())
	t.Equal(ba.Round(), vp.Round())
	t.Equal(ba.Stage(), vp.Stage())

	vrs := bb.loadVoteRecords(ba, false)
	t.NotNil(vrs)

	ib, found := vrs.ballots[ba.Node()]
	t.True(found)

	iba := ib.(ballot.INITBallotV0)
	t.True(ba.PreviousBlock().Equal(iba.PreviousBlock()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.PreviousRound(), iba.PreviousRound())
}

func (t *testBallotbox) TestINITVoteResultDraw() {
	bb := NewBallotbox(t.thresholdFunc(2, 67))

	// 2 ballot have the differnt previousBlock hash
	{
		ba := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node0"), nil, 0)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}
	{
		ba := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node1"), nil, 0)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultDraw, vp.Result())
		t.True(vp.IsFinished())
		t.NotNil(vp.FinishedAt())
		t.True(time.Now().Sub(vp.FinishedAt()) < time.Second)
	}

	{ // already finished
		ba := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node2"), nil, 0)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultDraw, vp.Result())
		t.True(vp.IsFinished())
		t.True(vp.IsClosed())
	}
}

func (t *testBallotbox) TestINITVoteResultMajority() {
	bb := NewBallotbox(t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()
	previousRound := base.Round(0)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node0"), previousBlock, previousRound)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node1"), previousBlock, previousRound)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}
	{
		vp, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(base.VoteResultMajority, vp.Result())
	}
}

func (t *testBallotbox) TestINITVoteproofClean() {
	bb := NewBallotbox(t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()
	previousRound := base.Round(0)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node0"), previousBlock, previousRound)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("node1"), previousBlock, previousRound)
	bar := t.newINITBallot(base.Height(9), base.Round(0), base.NewShortAddress("node0"), nil, 0)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}

	{
		_, err := bb.Vote(bar)
		t.NoError(err)
	}

	{
		vp, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(base.VoteResultMajority, vp.Result())
	}

	var remains []string
	bb.vrs.Range(func(k, v interface{}) bool {
		remains = append(remains, k.(string))
		return true
	})
	t.Equal(1, len(remains))

	var barFound bool
	for _, r := range remains {
		if r == "9-0-1" {
			barFound = true
			break
		}
	}
	t.False(barFound)
}

func (t *testBallotbox) newACCEPTBallot(
	height base.Height,
	round base.Round,
	node base.Address,
	proposal,
	newBlock valuehash.Hash,
) ballot.ACCEPTBallotV0 {
	vp := base.NewDummyVoteproof(
		height,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	if proposal == nil {
		proposal = valuehash.RandomSHA256()
	}
	if newBlock == nil {
		newBlock = valuehash.RandomSHA256()
	}

	ib := ballot.NewACCEPTBallotV0(
		node,
		height,
		round,
		proposal,
		newBlock,
		vp,
	)
	t.NoError(ib.Sign(t.pk, nil))

	return ib
}

func (t *testBallotbox) TestACCEPTVoteResultNotYet() {
	bb := NewBallotbox(t.thresholdFunc(2, 67))
	ba := t.newACCEPTBallot(base.Height(10), base.Round(0), base.NewShortAddress("test-for-accept-ballot"), nil, nil)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(base.VoteResultNotYet, vp.Result())

	t.Equal(ba.Height(), vp.Height())
	t.Equal(ba.Round(), vp.Round())
	t.Equal(ba.Stage(), vp.Stage())

	vrs := bb.loadVoteRecords(ba, false)
	t.NotNil(vrs)

	ib, found := vrs.ballots[ba.Node()]
	t.True(found)

	iba := ib.(ballot.ACCEPTBallotV0)
	t.True(ba.Proposal().Equal(iba.Proposal()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.NewBlock(), iba.NewBlock())
}

func (t *testBallotbox) TestACCEPTVoteResultDraw() {
	bb := NewBallotbox(t.thresholdFunc(2, 67))

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(base.Height(10), base.Round(0), base.NewShortAddress("node0"), nil, nil)
	ba1 := t.newACCEPTBallot(base.Height(10), base.Round(0), base.NewShortAddress("node1"), nil, nil)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}
	{
		vp, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(base.VoteResultDraw, vp.Result())
	}
}

func (t *testBallotbox) TestACCEPTVoteResultMajority() {
	bb := NewBallotbox(t.thresholdFunc(3, 66))

	proposal := valuehash.RandomSHA256()
	newBlock := valuehash.RandomSHA256()

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(base.Height(10), base.Round(0), base.NewShortAddress("node0"), proposal, newBlock)
	ba1 := t.newACCEPTBallot(base.Height(10), base.Round(0), base.NewShortAddress("node1"), proposal, newBlock)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}
	{
		vp, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(base.VoteResultMajority, vp.Result())
	}
}

func (t *testBallotbox) TestINITVoteResultMajorityClosed() {
	bb := NewBallotbox(t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()
	previousRound := base.Round(0)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("n0"), previousBlock, previousRound)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("n1"), previousBlock, previousRound)
	ba2 := t.newINITBallot(base.Height(10), base.Round(0), base.NewShortAddress("n2"), nil, 0)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
		t.False(vp.IsClosed())
	}

	{
		vp, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(base.VoteResultMajority, vp.Result())
		t.False(vp.IsClosed())
	}

	{
		vp, err := bb.Vote(ba2)
		t.NoError(err)
		t.Equal(base.VoteResultMajority, vp.Result())
		t.True(vp.IsClosed())
	}
}

func TestBallotbox(t *testing.T) {
	suite.Run(t, new(testBallotbox))
}
