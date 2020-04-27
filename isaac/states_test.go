package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
)

type testConsensusStates struct {
	baseTestStateHandler
}

func (t *testConsensusStates) TestINITVoteproofHigherHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	css := NewConsensusStates(t.localstate, nil, nil, nil, nil, nil, nil, nil)
	t.NotNil(css)

	initFact := ballot.NewINITBallotV0(
		nil,
		t.localstate.LastBlock().Height()+3,
		base.Round(2), // round is not important to go
		t.localstate.LastBlock().Hash(),
		t.localstate.LastBlock().Round(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	t.NoError(css.newVoteproof(vp))

	ctx := <-css.stateChan

	t.Equal(base.StateSyncing, ctx.toState)
	t.Equal(base.StageINIT, ctx.voteproof.Stage())
	t.Equal(initFact, ctx.voteproof.Majority())
}

func TestConsensusStates(t *testing.T) {
	suite.Run(t, new(testConsensusStates))
}
