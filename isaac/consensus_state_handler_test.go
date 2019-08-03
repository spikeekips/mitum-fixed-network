package isaac

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type testConsensusStateHandler struct {
	suite.Suite
}

func (t *testConsensusStateHandler) handler(suffrage Suffrage, timeoutWaitBallot time.Duration) (*ConsensusStateHandler, func()) {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)
	_ = homeState.SetBlock(nextBlock)

	thr, _ := NewThreshold(4, 67)
	cm := NewCompiler(homeState, NewBallotbox(thr))

	cn := t.newNetwork(homeState.Home())
	t.NoError(cn.Start())

	if suffrage == nil {
		suffrage = NewFixedProposerSuffrage(home, home)
	} else {
		suffrage.AddNodes(home)
	}

	pv := NewDummyProposalValidator()

	dp := NewDefaultProposalMaker(home, 0)
	cs, err := NewConsensusStateHandler(homeState, cm, cn, suffrage, pv, dp, timeoutWaitBallot)
	t.NoError(err)

	return cs, func() {
		_ = cs.Stop()
		_ = cn.Stop()
	}
}

func (t *testConsensusStateHandler) handlerActivated(
	suffrage Suffrage,
	timeoutWaitBallot time.Duration,
) (*ConsensusStateHandler, func(), VoteResult) {
	cs, closeFunc := t.handler(suffrage, timeoutWaitBallot)

	t.Equal(node.StateConsensus, cs.State())

	_ = cs.SetChanState(make(chan StateContext))

	t.NoError(cs.Start())
	defer cs.Stop()

	vr := NewVoteResult(
		cs.homeState.Block().Height().Add(1),
		cs.homeState.Block().Round()+1,
		StageINIT,
	).
		SetAgreement(Majority).
		SetBlock(cs.homeState.Block().Hash()).
		SetLastBlock(cs.homeState.PreviousBlock().Hash()).
		SetProposal(cs.homeState.Block().Proposal())

	t.NoError(cs.Activate(NewStateContext(node.StateConsensus).
		SetContext("vr", vr),
	))

	return cs, closeFunc, vr
}

func (t *testConsensusStateHandler) newNetwork(home node.Home) *network.ChannelNetwork {
	return network.NewChannelNetwork(
		home,
		func(sl seal.Seal) (seal.Seal, error) {
			return sl, xerrors.Errorf("echo back")
		},
	)
}

func (t *testConsensusStateHandler) TestNew() {
	defer common.DebugPanic()

	cs, closeFunc := t.handler(nil, time.Second*3)
	defer closeFunc()

	t.Equal(node.StateConsensus, cs.State())

	_ = cs.SetChanState(make(chan StateContext))

	t.NoError(cs.Start())
	defer cs.Stop()

	vr := NewVoteResult(
		cs.homeState.Block().Height().Add(1),
		cs.homeState.Block().Round()+1,
		StageINIT,
	).
		SetAgreement(Majority).
		SetBlock(cs.homeState.Block().Hash()).
		SetLastBlock(cs.homeState.PreviousBlock().Hash()).
		SetProposal(cs.homeState.Block().Proposal())

	t.NoError(cs.Activate(NewStateContext(node.StateConsensus).
		SetContext("vr", vr),
	))
}

func (t *testConsensusStateHandler) TestEmptyPreviousBlock() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()

	homeState := NewHomeState(home, lastBlock)

	thr, _ := NewThreshold(4, 67)
	cm := NewCompiler(homeState, NewBallotbox(thr))

	dp := NewDefaultProposalMaker(home, 0)
	_, err := NewConsensusStateHandler(homeState, cm, nil, nil, nil, dp, time.Second)
	t.Contains(err.Error(), "previous block is empty")
}

func (t *testConsensusStateHandler) TestBrodcastProposal() {
	cs, closeFunc, vr := t.handlerActivated(nil, time.Second*3)
	defer closeFunc()

	var proposal Proposal
	select {
	case <-time.After(time.Millisecond * 200):
		t.NoError(errors.New("timed out"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		var ok bool
		proposal, ok = message.(Proposal)
		t.True(ok)
	}

	t.Equal(ProposalType, proposal.Type())
	t.True(vr.Height().Equal(proposal.Height()))
	t.Equal(vr.Round(), proposal.Round())
	t.True(vr.Block().Equal(proposal.LastBlock()))
	t.True(
		cs.suffrage.Acting(
			vr.Height(),
			vr.Round(),
		).Proposer().Address().Equal(proposal.Proposer()),
	)
}

func (t *testConsensusStateHandler) TestTimeoutWaitProposal() {
	defer common.DebugPanic()

	proposer := node.NewRandomHome()
	suffrage := NewFixedProposerSuffrage(proposer)

	cs, closeFunc, vr := t.handlerActivated(suffrage, time.Millisecond*10)
	defer closeFunc()

	var ballot Ballot
	select {
	case <-time.After(time.Millisecond * 30):
		t.NoError(errors.New("timed out"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		var ok bool
		ballot, ok = message.(Ballot)
		t.True(ok)
		t.Equal(BallotType, ballot.Type())
	}

	// wait next round ballot
	t.Equal(StageINIT, ballot.Stage())
	t.True(vr.Height().Equal(ballot.Height()))
	t.Equal(vr.Round()+1, ballot.Round())
	t.True(vr.Block().Equal(ballot.Block()))
	t.True(vr.Proposal().Equal(ballot.Proposal()))
}

func (t *testConsensusStateHandler) TestReceiveProposalAndNestStages() {
	cs, closeFunc, vr := t.handlerActivated(nil, time.Second*3)
	defer closeFunc()

	cs.compiler.lastINITVoteResult = vr

	select {
	case <-time.After(time.Millisecond * 50):
		t.NoError(errors.New("timed out; wait proposal"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		proposal, ok := message.(Proposal)
		t.True(ok)

		t.Equal(ProposalType, proposal.Type())
		t.True(vr.Height().Equal(proposal.Height()))
		t.Equal(vr.Round(), proposal.Round())
		t.True(cs.homeState.Block().Hash().Equal(proposal.LastBlock()))
		t.True(cs.homeState.Home().Address().Equal(proposal.Proposer()))

		err := cs.ReceiveProposal(proposal)
		t.NoError(err)
	}

	// wait sign ballot
	select {
	case <-time.After(time.Millisecond * 50):
		t.NoError(errors.New("timed out; sign ballot"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		ballot, ok := message.(Ballot)
		t.True(ok)

		t.Equal(BallotType, ballot.Type())
		t.Equal(StageSIGN, ballot.Stage())
		t.True(vr.Height().Equal(ballot.Height()))
		t.Equal(vr.Round(), ballot.Round())
		t.True(cs.homeState.Home().Address().Equal(ballot.Node()))
		t.NotEmpty(ballot.Block())
	}
}

func (t *testConsensusStateHandler) TestProposalTimeoutNextRound() {
	proposer := node.NewRandomHome()
	suffrage := NewFixedProposerSuffrage(proposer)
	cs, closeFunc, vr := t.handlerActivated(suffrage, time.Millisecond*50)
	defer closeFunc()

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.New("timed out; wait next round ballot"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		ballot, ok := message.(Ballot)
		t.True(ok)

		t.Equal(BallotType, ballot.Type())
		t.Equal(StageINIT, ballot.Stage())
		t.True(vr.Height().Equal(ballot.Height()))
		t.Equal(vr.Round()+1, ballot.Round())
		t.True(cs.homeState.Home().Address().Equal(ballot.Node()))
		t.NotEmpty(ballot.Block())
	}
}

func (t *testConsensusStateHandler) TestBallotTimeoutNextRound() {
	cs, closeFunc, vr := t.handlerActivated(nil, time.Millisecond*50)
	defer closeFunc()

	cs.compiler.lastINITVoteResult = vr

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.New("timed out; wait proposal"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		proposal, ok := message.(Proposal)
		t.True(ok)

		t.Equal(ProposalType, proposal.Type())
		t.True(vr.Height().Equal(proposal.Height()))
		t.Equal(vr.Round(), proposal.Round())
		t.True(cs.homeState.Block().Hash().Equal(proposal.LastBlock()))
		t.True(cs.homeState.Home().Address().Equal(proposal.Proposer()))

		err := cs.ReceiveProposal(proposal)
		t.NoError(err)
	}

	// wait sign ballot
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.New("timed out; sign ballot"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		ballot, ok := message.(Ballot)
		t.True(ok)

		t.Equal(BallotType, ballot.Type())
		t.Equal(StageSIGN, ballot.Stage())
		t.True(vr.Height().Equal(ballot.Height()))
		t.Equal(vr.Round(), ballot.Round())
		t.True(cs.homeState.Home().Address().Equal(ballot.Node()))
		t.NotEmpty(ballot.Block())
	}

	// wait next ballot, accept
	select {
	case <-time.After(time.Second):
		t.NoError(errors.New("timed out; sign ballot"))
		return
	case message := <-cs.nt.(*network.ChannelNetwork).Reader():
		ballot, ok := message.(Ballot)
		t.True(ok)

		t.Equal(BallotType, ballot.Type())
		t.Equal(StageINIT, ballot.Stage())
		t.True(vr.Height().Equal(ballot.Height()))
		t.Equal(vr.Round()+1, ballot.Round())
		t.True(cs.homeState.Home().Address().Equal(ballot.Node()))
		t.NotEmpty(ballot.Block())
	}

	closeFunc()
}

func TestConsensusStateHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateHandler))
}
