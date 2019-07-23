package isaac

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type testJoinStateHandler struct {
	suite.Suite
}

func (t *testJoinStateHandler) handler(
	intervalBroadcastINITBallot, timeoutWaitVoteResult time.Duration,
) (*JoinStateHandler, func()) {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)
	_ = homeState.SetBlock(nextBlock)

	thr, _ := NewThreshold(4, 67)
	cm := NewCompiler(homeState, NewBallotbox(thr))

	cn := t.newNetwork(homeState.Home())
	t.NoError(cn.Start())

	js, err := NewJoinStateHandler(homeState, cm, cn, intervalBroadcastINITBallot, timeoutWaitVoteResult)
	t.NoError(err)

	return js, func() {
		cn.Stop()
	}
}

func (t *testJoinStateHandler) newNetwork(home node.Home) *network.ChannelNetwork {
	return network.NewChannelNetwork(
		home,
		func(sl seal.Seal) (seal.Seal, error) {
			return sl, xerrors.Errorf("echo back")
		},
	)
}

func (t *testJoinStateHandler) TestNew() {
	js, closeFunc := t.handler(time.Second*3, time.Second*6)
	defer closeFunc()

	_ = js.SetChanState(make(chan node.State))

	err := js.Start()
	t.NoError(err)

	// check timer run count
	<-time.After(time.Millisecond * 20)
	t.True(js.timer.RunCount() > 0)
}

func (t *testJoinStateHandler) TestEmptyPreviousBlock() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()

	homeState := NewHomeState(home, lastBlock)

	thr, _ := NewThreshold(4, 67)
	cm := NewCompiler(homeState, NewBallotbox(thr))

	_, err := NewJoinStateHandler(homeState, cm, nil, time.Second*10, time.Second*20)
	t.Contains(err.Error(), "previous block is empty")
}

func (t *testJoinStateHandler) TestBroadcastINITBallot() {
	js, closeFunc := t.handler(time.Second*3, time.Second*6)
	defer closeFunc()

	_ = js.SetChanState(make(chan node.State))

	err := js.Start()
	t.NoError(err)

	// check timer run count
	var ballot Ballot
	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(errors.New("timed out"))
	case message := <-js.nt.(*network.ChannelNetwork).Reader():
		var ok bool
		ballot, ok = message.(Ballot)
		t.True(ok)
	}

	t.Equal(BallotType, ballot.Type())
	t.Equal(StageINIT, ballot.Stage())
	t.True(js.homeState.Home().Address().Equal(ballot.Node()))
	t.True(js.homeState.Block().Height().Add(1).Equal(ballot.Height()))
	t.Equal(js.homeState.Block().Round()+1, ballot.Round())
	t.True(js.homeState.Block().Hash().Equal(ballot.Block()))
	t.True(js.homeState.PreviousBlock().Hash().Equal(ballot.LastBlock()))
	t.True(js.homeState.Block().Proposal().Equal(ballot.Proposal()))
}

func (t *testJoinStateHandler) TestReceiveVoteResult() {
}

func (t *testJoinStateHandler) TestTimeoutVoteProof() {
	js, closeFunc := t.handler(time.Second*1, time.Millisecond*20)
	defer closeFunc()

	chRequest := make(chan seal.Seal)
	js.nt.(*network.ChannelNetwork).SetHandler(func(sl seal.Seal) (seal.Seal, error) {
		chRequest <- sl
		return nil, nil
	})

	{ // register other node
		home := node.NewRandomHome()
		js.nt.(*network.ChannelNetwork).AddMembers(t.newNetwork(home))
	}

	_ = js.SetChanState(make(chan node.State))

	err := js.Start()
	t.NoError(err)

	sl := <-chRequest

	rv, ok := sl.(Request)
	t.True(ok)

	t.Equal(RquestType, rv.Type())
	t.Equal(RequestVoteProof, rv.Request())
}

func TestJoinStateHandler(t *testing.T) {
	suite.Run(t, new(testJoinStateHandler))
}
