package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testConsensusStates struct {
	baseTestStateHandler
}

func (t *testConsensusStates) TestINITVoteproofHigherHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	css := NewConsensusStates(t.localstate, nil, nil, nil, nil, nil, nil, nil)
	t.NotNil(css)

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localstate.LastBlock().Height() + 3,
			round:  Round(2), // round is not important to go
		},
		previousBlock: t.localstate.LastBlock().Hash(),
		previousRound: t.localstate.LastBlock().Round(),
	}

	vp, err := t.newVoteproof(StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	t.NoError(css.newVoteproof(vp))

	ctx := <-css.stateChan

	t.Equal(StateSyncing, ctx.toState)
	t.Equal(StageINIT, ctx.voteproof.Stage())
	t.Equal(initFact, ctx.voteproof.Majority())
}

func TestConsensusStates(t *testing.T) {
	suite.Run(t, new(testConsensusStates))
}
