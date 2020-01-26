package mitum

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

type testBallotbox struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotbox) SetupSuite() {
	_ = hint.RegisterType(INITBallotType, "init-ballot")
	_ = hint.RegisterType(ProposalBallotType, "proposal")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType((valuehash.SHA256{}).Hint().Type(), "sha256")
}

func (t *testBallotbox) TestNew() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)
	ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("test-for-init-ballot"))

	vr, err := bb.Vote(ba)
	t.NoError(err)
	t.NotEmpty(vr)
}

func (t *testBallotbox) newINITBallot(height Height, round Round, node Address) Ballot {
	return INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			height: height,
			round:  round,
			node:   node,
		},
		previousBlock: valuehash.RandomSHA256(),
		previousRound: Round(0),
	}
}

func (t *testBallotbox) TestVoteRace() {
	threshold, _ := NewThreshold(50, 100)
	bb := NewBallotbox(threshold)

	checkDone := make(chan bool)
	vrChan := make(chan interface{}, 49)

	go func() {
		for i := range vrChan {
			switch c := i.(type) {
			case error:
				t.NoError(c)
			case VoteResult:
				t.Equal(VoteResultNotYet, c.Result())
			}
		}
		checkDone <- true
	}()

	var wg sync.WaitGroup
	wg.Add(49)
	for i := 0; i < 49; i++ {
		go func() {
			defer wg.Done()
			ba := t.newINITBallot(Height(10), Round(0), RandomShortAddress())

			vr, err := bb.Vote(ba)
			if err != nil {
				vrChan <- err
			} else {
				vrChan <- vr
			}
		}()
	}
	wg.Wait()
	close(vrChan)

	<-checkDone
}

func (t *testBallotbox) TestINITVoteResultNotYet() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)
	ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("test-for-init-ballot"))

	vr, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(VoteResultNotYet, vr.Result())

	t.Equal(ba.Height(), vr.Height())
	t.Equal(ba.Round(), vr.Round())
	t.Equal(ba.Stage(), vr.Stage())

	vrc, found := bb.VoteRecord(ba)
	t.True(found)
	ivrc := vrc.(VoteRecordINIT)
	t.True(ba.(INITBallot).PreviousBlock().Equal(ivrc.previousBlock))
	t.Equal(ba.Node(), ivrc.node)
	t.Equal(ba.(INITBallot).PreviousRound(), ivrc.previousRound)
}

func (t *testBallotbox) TestINITVoteResultDraw() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	{
		ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("node0"))
		vr, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("node1"))
		vr, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(VoteResultDraw, vr.Result())
		t.True(vr.IsFinished())
	}

	{ // already finished
		ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("node2"))
		vr, err := bb.Vote(ba)
		t.NoError(err)
		t.Equal(VoteResultDraw, vr.Result())
		t.True(vr.IsFinished())
	}
}

func (t *testBallotbox) TestINITVoteResultMajority() {
	threshold, _ := NewThreshold(3, 66)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newINITBallot(Height(10), Round(0), NewShortAddress("node0"))
	ba1 := t.newINITBallot(Height(10), Round(0), NewShortAddress("node1"))

	{ // set same previousBlock and previousRound
		ib0 := ba0.(INITBallotV0)
		ib1 := ba1.(INITBallotV0)
		ib1.previousBlock = ib0.previousBlock
		ib1.previousRound = ib0.previousRound

		ba0 = ib0
		ba1 = ib1
	}

	{
		vr, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		vr, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(VoteResultMajority, vr.Result())
	}
}

func (t *testBallotbox) newSIGNBallot(height Height, round Round, node Address) Ballot {
	return SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			height: height,
			round:  round,
			node:   node,
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}
}

func (t *testBallotbox) TestSIGNVoteResultNotYet() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)
	ba := t.newSIGNBallot(Height(10), Round(0), NewShortAddress("test-for-sign-ballot"))

	vr, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(VoteResultNotYet, vr.Result())

	t.Equal(ba.Height(), vr.Height())
	t.Equal(ba.Round(), vr.Round())
	t.Equal(ba.Stage(), vr.Stage())

	vrc, found := bb.VoteRecord(ba)
	t.True(found)

	ivrc := vrc.(VoteRecordSIGN)
	t.True(ba.(SIGNBallot).Proposal().Equal(ivrc.proposal))
	t.Equal(ba.Node(), ivrc.node)
	t.Equal(ba.(SIGNBallot).NewBlock(), ivrc.newBlock)
}

func (t *testBallotbox) TestSIGNVoteResultDraw() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newSIGNBallot(Height(10), Round(0), NewShortAddress("node0"))
	ba1 := t.newSIGNBallot(Height(10), Round(0), NewShortAddress("node1"))

	{
		vr, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		vr, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(VoteResultDraw, vr.Result())
	}
}

func (t *testBallotbox) TestSIGNVoteResultMajority() {
	threshold, _ := NewThreshold(3, 66)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newSIGNBallot(Height(10), Round(0), NewShortAddress("node0"))
	ba1 := t.newSIGNBallot(Height(10), Round(0), NewShortAddress("node1"))

	{ // set same previousBlock and previousRound
		ib0 := ba0.(SIGNBallotV0)
		ib1 := ba1.(SIGNBallotV0)
		ib1.proposal = ib0.proposal
		ib1.newBlock = ib0.newBlock

		ba0 = ib0
		ba1 = ib1
	}

	{
		vr, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		vr, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(VoteResultMajority, vr.Result())
	}
}

func (t *testBallotbox) newACCEPTBallot(height Height, round Round, node Address) Ballot {
	return ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			height: height,
			round:  round,
			node:   node,
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}
}

func (t *testBallotbox) TestACCEPTVoteResultNotYet() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)
	ba := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("test-for-accept-ballot"))

	vr, err := bb.Vote(ba)
	t.NoError(err)
	t.Equal(VoteResultNotYet, vr.Result())

	t.Equal(ba.Height(), vr.Height())
	t.Equal(ba.Round(), vr.Round())
	t.Equal(ba.Stage(), vr.Stage())

	vrc, found := bb.VoteRecord(ba)
	t.True(found)
	ivrc := vrc.(VoteRecordACCEPT)
	t.True(ba.(ACCEPTBallot).Proposal().Equal(ivrc.proposal))
	t.Equal(ba.Node(), ivrc.node)
	t.Equal(ba.(ACCEPTBallot).NewBlock(), ivrc.newBlock)
}

func (t *testBallotbox) TestACCEPTVoteResultDraw() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("node0"))
	ba1 := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("node1"))

	{
		vr, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		vr, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(VoteResultDraw, vr.Result())
	}
}

func (t *testBallotbox) TestACCEPTVoteResultMajority() {
	threshold, _ := NewThreshold(3, 66)
	bb := NewBallotbox(threshold)

	// 2 ballot have the differnt previousBlock hash
	ba0 := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("node0"))
	ba1 := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("node1"))

	{ // set same previousBlock and previousRound
		ib0 := ba0.(ACCEPTBallotV0)
		ib1 := ba1.(ACCEPTBallotV0)
		ib1.proposal = ib0.proposal
		ib1.newBlock = ib0.newBlock

		ba0 = ib0
		ba1 = ib1
	}

	{
		vr, err := bb.Vote(ba0)
		t.NoError(err)
		t.Equal(VoteResultNotYet, vr.Result())
	}
	{
		vr, err := bb.Vote(ba1)
		t.NoError(err)
		t.Equal(VoteResultMajority, vr.Result())
	}
}

func (t *testBallotbox) TestGetVoteRecord() {
	threshold, _ := NewThreshold(3, 66)
	bb := NewBallotbox(threshold)

	{
		ba := t.newACCEPTBallot(Height(10), Round(0), NewShortAddress("node0"))
		_, err := bb.Vote(ba)
		t.NoError(err)

		_, isVoted := bb.VoteRecord(ba)
		t.True(isVoted)
		_, found := bb.vrs.Load(bb.vrsKey(ba))
		t.True(found)
	}

	{
		ba := t.newACCEPTBallot(Height(11), Round(0), NewShortAddress("node1"))
		_, isVoted := bb.VoteRecord(ba)
		t.False(isVoted)
		_, found := bb.vrs.Load(bb.vrsKey(ba))
		t.False(found)
	}
}

func TestBallotbox(t *testing.T) {
	suite.Run(t, new(testBallotbox))
}
