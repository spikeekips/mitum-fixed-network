package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/node"
)

type testBootingStateHandler struct {
	suite.Suite
}

func (t *testBootingStateHandler) TestMoveToNextState() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()

	homeState := NewHomeState(home, lastBlock)

	chanState := make(chan node.State)
	bs := NewBootingStateHandler(homeState)
	_ = bs.SetChanState(chanState)

	err := bs.Start()
	t.NoError(err)

	nextState := <-chanState
	t.Equal(node.StateJoin, nextState)
}
func TestBootingStateHandler(t *testing.T) {
	suite.Run(t, new(testBootingStateHandler))
}
