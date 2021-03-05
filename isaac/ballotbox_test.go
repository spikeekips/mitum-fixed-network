package isaac

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/key"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotbox struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testBallotbox) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotbox) thresholdFunc(total uint, ratio float64) func() base.Threshold {
	localNode := channetwork.RandomLocalNode(util.UUID().String())
	ls, err := NewLocal(nil, nil, localNode, TestNetworkID)
	t.NoError(err)
	t.NoError(ls.Initialize())

	r := base.ThresholdRatio(ratio)
	_ = ls.Policy().SetThresholdRatio(r)

	threshold, _ := base.NewThreshold(total, r)

	return func() base.Threshold {
		return threshold
	}
}

func (t *testBallotbox) suffragesFunc(n ...base.Address) func() []base.Address {
	return func() []base.Address {
		return n
	}
}

func (t *testBallotbox) TestNew() {
	n := base.RandomStringAddress()
	bb := NewBallotbox(t.suffragesFunc(n), t.thresholdFunc(2, 67))
	ba := t.newINITBallot(base.Height(10), base.Round(0), n, nil)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.NotEmpty(vp)
}

func (t *testBallotbox) TestNotInSuffrage() {
	n := base.RandomStringAddress()
	bb := NewBallotbox(t.suffragesFunc(n), t.thresholdFunc(2, 67))

	other := base.RandomStringAddress()
	ba := t.newINITBallot(base.Height(10), base.Round(0), other, nil)

	_, err := bb.Vote(ba)
	t.Contains(err.Error(), "not in suffrages")
}

func (t *testBallotbox) newINITBallot(
	height base.Height,
	round base.Round,
	node base.Address,
	previousBlock valuehash.Hash,
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
		vp,
		vp,
	)
	t.NoError(ib.Sign(t.pk, nil))

	return ib
}

func (t *testBallotbox) TestVoteRace() {
	node := base.RandomStringAddress()
	bb := NewBallotbox(t.suffragesFunc(node), t.thresholdFunc(50, 100))

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
			ba := t.newINITBallot(base.Height(10), base.Round(0), node, nil)

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
	node := base.RandomStringAddress()
	bb := NewBallotbox(t.suffragesFunc(node), t.thresholdFunc(2, 67))
	ba := t.newINITBallot(base.Height(10), base.Round(0), node, nil)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(base.VoteResultNotYet, vp.Result())

	t.Equal(ba.Height(), vp.Height())
	t.Equal(ba.Round(), vp.Round())
	t.Equal(ba.Stage(), vp.Stage())

	vrs := bb.loadVoteRecords(ba, false)
	t.NotNil(vrs)

	ib, found := vrs.ballots[ba.Node().String()]
	t.True(found)

	iba := ib.(ballot.INITBallotV0)
	t.True(ba.PreviousBlock().Equal(iba.PreviousBlock()))
	t.Equal(ba.Node(), iba.Node())
}

func (t *testBallotbox) TestINITVoteResultDraw() {
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(2, 67))

	// 2 ballot have the differnt previousBlock hash
	{
		ba := t.newINITBallot(base.Height(10), base.Round(0), nodes[0], nil)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}
	{
		ba := t.newINITBallot(base.Height(10), base.Round(0), nodes[1], nil)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultDraw, vp.Result())
		t.True(vp.IsFinished())
		t.NotNil(vp.FinishedAt())
		t.True(time.Since(vp.FinishedAt()) < time.Second)
	}

	{ // already finished
		ba := t.newINITBallot(base.Height(10), base.Round(0), nodes[2], nil)
		vp, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(base.VoteResultDraw, vp.Result())
		t.True(vp.IsFinished())
		t.True(vp.IsClosed())
	}
}

func (t *testBallotbox) TestINITVoteResultMajority() {
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), nodes[0], previousBlock)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), nodes[1], previousBlock)

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
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), nodes[0], previousBlock)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), nodes[1], previousBlock)
	bar := t.newINITBallot(base.Height(9), base.Round(0), nodes[0], nil)

	{
		vp, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(base.VoteResultNotYet, vp.Result())
	}

	{
		_, err := bb.Vote(bar)
		t.NoError(err)
	}

	vp, err := bb.Vote(ba1)
	t.NoError(err)
	t.Equal(base.VoteResultMajority, vp.Result())

	t.NoError(bb.Clean(vp.Height() - 1))

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
	node := base.RandomStringAddress()
	bb := NewBallotbox(t.suffragesFunc(node), t.thresholdFunc(2, 67))
	ba := t.newACCEPTBallot(base.Height(10), base.Round(0), node, nil, nil)

	vp, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(base.VoteResultNotYet, vp.Result())

	t.Equal(ba.Height(), vp.Height())
	t.Equal(ba.Round(), vp.Round())
	t.Equal(ba.Stage(), vp.Stage())

	vrs := bb.loadVoteRecords(ba, false)
	t.NotNil(vrs)

	ib, found := vrs.ballots[ba.Node().String()]
	t.True(found)

	iba := ib.(ballot.ACCEPTBallotV0)
	t.True(ba.Proposal().Equal(iba.Proposal()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.NewBlock(), iba.NewBlock())
}

func (t *testBallotbox) TestACCEPTVoteResultDraw() {
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(2, 67))

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(base.Height(10), base.Round(0), nodes[0], nil, nil)
	ba1 := t.newACCEPTBallot(base.Height(10), base.Round(0), nodes[1], nil, nil)

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
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(3, 66))

	proposal := valuehash.RandomSHA256()
	newBlock := valuehash.RandomSHA256()

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(base.Height(10), base.Round(0), nodes[0], proposal, newBlock)
	ba1 := t.newACCEPTBallot(base.Height(10), base.Round(0), nodes[1], proposal, newBlock)

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
	nodes := []base.Address{
		base.RandomStringAddress(),
		base.RandomStringAddress(),
		base.RandomStringAddress(),
	}
	bb := NewBallotbox(t.suffragesFunc(nodes...), t.thresholdFunc(3, 66))

	previousBlock := valuehash.RandomSHA256()

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(base.Height(10), base.Round(0), nodes[0], previousBlock)
	ba1 := t.newINITBallot(base.Height(10), base.Round(0), nodes[1], previousBlock)
	ba2 := t.newINITBallot(base.Height(10), base.Round(0), nodes[2], nil)

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

func (t *testBallotbox) TestLatestBallot() {
	node := base.RandomStringAddress()

	ba0 := t.newINITBallot(base.Height(10), base.Round(0), node, nil)
	ba1 := t.newINITBallot(base.Height(10), base.Round(1), node, nil)
	ba2 := t.newINITBallot(base.Height(11), base.Round(0), node, nil)
	ba3 := t.newINITBallot(base.Height(10), base.Round(1), node, nil)

	bb := NewBallotbox(t.suffragesFunc(node), t.thresholdFunc(3, 66))

	_, err := bb.Vote(ba0)
	t.NoError(err)
	t.True(bb.LatestBallot().Hash().Equal(ba0.Hash()))

	_, err = bb.Vote(ba1)
	t.NoError(err)
	t.True(bb.LatestBallot().Hash().Equal(ba1.Hash()))

	_, err = bb.Vote(ba2)
	t.NoError(err)
	t.True(bb.LatestBallot().Hash().Equal(ba2.Hash()))

	_, err = bb.Vote(ba3)
	t.NoError(err)
	t.True(bb.LatestBallot().Hash().Equal(ba2.Hash()))
}

func TestBallotbox(t *testing.T) {
	suite.Run(t, new(testBallotbox))
}
