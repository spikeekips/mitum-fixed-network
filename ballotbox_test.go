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

	pk key.Privatekey
}

func (t *testBallotbox) SetupSuite() {
	_ = hint.RegisterType(INITBallotType, "init-ballot")
	_ = hint.RegisterType(ProposalBallotType, "proposal")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType((valuehash.SHA256{}).Hint().Type(), "sha256")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotbox) TestNew() {
	threshold, _ := NewThreshold(2, 67)
	bb := NewBallotbox(threshold)
	ba := t.newINITBallot(Height(10), Round(0), NewShortAddress("test-for-init-ballot"))

	vr, err := bb.Vote(ba)
	t.NoError(err)
	t.NotEmpty(vr)
}

func (t *testBallotbox) newINITBallot(height Height, round Round, node Address) INITBallotV0 {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		INITBallotV0Fact: INITBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: height,
				round:  round,
			},
			previousBlock: valuehash.RandomSHA256(),
			previousRound: Round(0),
		},
	}
	t.NoError(ib.Sign(t.pk, nil))

	return ib
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

	ib, found := vr.ballots[ba.Node()]
	t.True(found)

	iba := ib.(INITBallotV0)
	t.True(ba.PreviousBlock().Equal(iba.PreviousBlock()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.PreviousRound(), iba.PreviousRound())
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
		ba1.previousBlock = ba0.previousBlock
		ba1.previousRound = ba0.previousRound

		t.NoError(ba1.Sign(t.pk, nil))
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

func (t *testBallotbox) newSIGNBallot(height Height, round Round, node Address) SIGNBallotV0 {
	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		SIGNBallotV0Fact: SIGNBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: height,
				round:  round,
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}
	t.NoError(ib.Sign(t.pk, nil))

	return ib
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

	ib, found := vr.ballots[ba.Node()]
	t.True(found)

	iba := ib.(SIGNBallotV0)
	t.True(ba.Proposal().Equal(iba.Proposal()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.NewBlock(), iba.NewBlock())
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
		ba1.proposal = ba0.proposal
		ba1.newBlock = ba0.newBlock

		t.NoError(ba1.Sign(t.pk, nil))
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

func (t *testBallotbox) newACCEPTBallot(height Height, round Round, node Address) ACCEPTBallotV0 {
	ib := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		ACCEPTBallotV0Fact: ACCEPTBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: height,
				round:  round,
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}
	t.NoError(ib.Sign(t.pk, nil))

	return ib
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

	ib, found := vr.ballots[ba.Node()]
	t.True(found)

	iba := ib.(ACCEPTBallotV0)
	t.True(ba.Proposal().Equal(iba.Proposal()))
	t.Equal(ba.Node(), iba.Node())
	t.Equal(ba.NewBlock(), iba.NewBlock())
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
		ba1.proposal = ba0.proposal
		ba1.newBlock = ba0.newBlock

		t.NoError(ba1.Sign(t.pk, nil))
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

func TestBallotbox(t *testing.T) {
	suite.Run(t, new(testBallotbox))
}
