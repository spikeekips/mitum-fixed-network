package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type testBallotbox struct {
	suite.Suite
}

func (t *testBallotbox) vote(
	bb *Ballotbox,
	n node.Address,
	stage Stage,
	lastBlock,
	nextBlock Block,
) (VoteResult, error) {
	return bb.Vote(
		n,
		nextBlock.Height(),
		nextBlock.Round(),
		stage,
		nextBlock.Hash(),
		lastBlock.Hash(),
		nextBlock.Proposal(),
	)
}

func (t *testBallotbox) TestVote() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	vr, err := t.vote(bb, home.Address(), StageINIT, lastBlock, nextBlock)
	t.NoError(err)

	t.False(vr.IsFinished())
	t.False(vr.IsClosed())
	t.True(nextBlock.Height().Equal(vr.Height()))
	t.Equal(nextBlock.Round(), vr.Round())
	t.Equal(StageINIT, vr.Stage())
}

func (t *testBallotbox) TestGotMajority() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	// not yet over threshold
	_, threshold := thr.Get(StageSIGN)
	for i := uint(0); i < threshold-1; i++ {
		home := node.NewRandomHome()

		var vr VoteResult
		vr, err = t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
		t.NoError(err)

		t.False(vr.IsFinished())
		t.False(vr.IsClosed())
		t.True(nextBlock.Height().Equal(vr.Height()))
		t.Equal(nextBlock.Round(), vr.Round())
		t.Equal(StageSIGN, vr.Stage())
	}

	// over threshold
	home := node.NewRandomHome()
	vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
	t.NoError(err)

	t.True(vr.IsFinished())
	t.False(vr.IsClosed())
	t.False(vr.GotDraw())
	t.True(vr.GotMajority())
	t.True(nextBlock.Height().Equal(vr.Height()))
	t.Equal(nextBlock.Round(), vr.Round())
	t.Equal(StageSIGN, vr.Stage())

	t.Equal(int(threshold), len(vr.Records()))
}

func (t *testBallotbox) TestClosed() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	// vote by threshold
	_, threshold := thr.Get(StageSIGN)
	for i := uint(0); i < threshold; i++ {
		home := node.NewRandomHome()
		var vr VoteResult
		vr, err = t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
		t.NoError(err)

		if i == threshold-1 {
			t.True(vr.IsFinished())
		} else {
			t.False(vr.IsFinished())
		}
		t.False(vr.IsClosed())
		t.True(nextBlock.Height().Equal(vr.Height()))
		t.Equal(nextBlock.Round(), vr.Round())
		t.Equal(StageSIGN, vr.Stage())
	}

	// one more vote
	home := node.NewRandomHome()
	vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
	t.NoError(err)

	t.True(vr.IsFinished())
	t.True(vr.IsClosed())
	t.False(vr.GotDraw())
	t.True(vr.GotMajority())
	t.True(nextBlock.Height().Equal(vr.Height()))
	t.Equal(nextBlock.Round(), vr.Round())
	t.Equal(StageSIGN, vr.Stage())

	t.Equal(int(threshold)+1, len(vr.Records()))
}

func (t *testBallotbox) TestGotDrawAnotherNextBlock() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	// vote by half
	total, threshold := thr.Get(StageSIGN)
	for i := uint(0); i < threshold-1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
		t.NoError(err)

		t.False(vr.IsFinished())
		t.False(vr.IsClosed())
		t.True(nextBlock.Height().Equal(vr.Height()))
		t.Equal(nextBlock.Round(), vr.Round())
		t.Equal(StageSIGN, vr.Stage())
	}

	// vote by left with another next block
	anotherNextBlock := NewRandomNextBlock(lastBlock)

	var lastVR VoteResult
	for i := uint(0); i < total-threshold+1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, anotherNextBlock)
		t.NoError(err)

		lastVR = vr
	}

	t.True(lastVR.IsFinished())
	t.False(lastVR.IsClosed())
	t.True(lastVR.GotDraw())
	t.False(lastVR.GotMajority())
	t.True(nextBlock.Height().Equal(lastVR.Height()))
	t.Equal(nextBlock.Round(), lastVR.Round())
	t.Equal(StageSIGN, lastVR.Stage())

	t.Equal(int(total), len(lastVR.Records()))
}

func (t *testBallotbox) TestGotDrawAnotherProposal() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	// vote by half
	total, threshold := thr.Get(StageSIGN)
	for i := uint(0); i < threshold-1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
		t.NoError(err)

		t.False(vr.IsFinished())
		t.False(vr.IsClosed())
		t.True(nextBlock.Height().Equal(vr.Height()))
		t.Equal(nextBlock.Round(), vr.Round())
		t.Equal(StageSIGN, vr.Stage())
	}

	// vote by left with same next block hash, but another proposal
	anotherNextBlock := NewRandomNextBlock(lastBlock)
	anotherNextBlock.hash = nextBlock.Hash()

	var lastVR VoteResult
	for i := uint(0); i < total-threshold+1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, anotherNextBlock)
		t.NoError(err)

		lastVR = vr
	}

	t.True(lastVR.IsFinished())
	t.False(lastVR.IsClosed())
	t.True(lastVR.GotDraw())
	t.False(lastVR.GotMajority())
	t.True(nextBlock.Height().Equal(lastVR.Height()))
	t.Equal(nextBlock.Round(), lastVR.Round())
	t.Equal(StageSIGN, lastVR.Stage())

	t.Equal(int(total), len(lastVR.Records()))
}

func (t *testBallotbox) TestGotDrawAnotherLastBlock() {
	defer common.DebugPanic()

	thr, err := NewThreshold(4, 67)
	t.NoError(err)

	bb := NewBallotbox(thr)

	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	// vote by half
	total, threshold := thr.Get(StageSIGN)
	for i := uint(0); i < threshold-1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, lastBlock, nextBlock)
		t.NoError(err)

		t.False(vr.IsFinished())
		t.False(vr.IsClosed())
		t.True(nextBlock.Height().Equal(vr.Height()))
		t.Equal(nextBlock.Round(), vr.Round())
		t.Equal(StageSIGN, vr.Stage())
	}

	// vote by left with same next block hash, but another proposal
	anotherLastBlock := lastBlock
	anotherLastBlock.hash = NewRandomBlockHash()

	var lastVR VoteResult
	for i := uint(0); i < total-threshold+1; i++ {
		home := node.NewRandomHome()
		vr, err := t.vote(bb, home.Address(), StageSIGN, anotherLastBlock, nextBlock)
		t.NoError(err)

		lastVR = vr
	}

	t.True(lastVR.IsFinished())
	t.False(lastVR.IsClosed())
	t.True(lastVR.GotDraw())
	t.False(lastVR.GotMajority())
	t.True(nextBlock.Height().Equal(lastVR.Height()))
	t.Equal(nextBlock.Round(), lastVR.Round())
	t.Equal(StageSIGN, lastVR.Stage())

	t.Equal(int(total), len(lastVR.Records()))
}

func TestBallotbox(t *testing.T) {
	suite.Run(t, new(testBallotbox))
}
