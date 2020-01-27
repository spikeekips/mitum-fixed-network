package mitum

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateJoiningHandler struct {
	suite.Suite

	policy     *LocalPolicy
	localNode  *LocalNode
	remoteNode *LocalNode
	localState *LocalState
}

func (t *testConsensusStateJoiningHandler) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")

	t.localNode = RandomLocalNode("local", nil)
	t.policy = NewLocalPolicy()
	t.localState = NewLocalState(t.localNode, t.policy)

	t.remoteNode = RandomLocalNode("remote", nil)
	t.NoError(t.localState.Nodes().Add(t.remoteNode))
}

func (t *testConsensusStateJoiningHandler) TestNew() {
	cs, err := NewConsensusStateJoiningHandler(t.localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
}

func (t *testConsensusStateJoiningHandler) TestKeepBroadcastingINITBallot() {
	policy := NewLocalPolicy()
	localState := NewLocalState(t.localNode, policy)
	t.NoError(localState.Nodes().Add(t.remoteNode))
	localState.
		SetLastBlockHeight(Height(33)).
		SetLastBlockRound(Round(3)).
		SetLastBlockHash(valuehash.RandomSHA256())

	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)
	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	time.Sleep(time.Millisecond * 50)

	received := <-t.remoteNode.Channel().ReceiveSeal()
	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(INITBallotV0{}, received)

	ballot := received.(INITBallotV0)

	t.NoError(ballot.IsValid(nil))

	t.True(localState.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(StageINIT, ballot.Stage())
	t.Equal(localState.LastBlockHeight()+1, ballot.Height())
	t.Equal(Round(0), ballot.Round())
	t.True(localState.Node().Address().Equal(ballot.Node()))
	t.True(localState.LastBlockHash().Equal(ballot.PreviousBlock()))
	t.Equal(localState.LastBlockRound(), ballot.PreviousRound())
}

func TestConsensusStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateJoiningHandler))
}
