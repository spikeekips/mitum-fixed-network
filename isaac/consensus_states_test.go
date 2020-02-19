package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testConsensusStates struct {
	baseTestConsensusStateHandler
}

func (t *testConsensusStates) TestINITVoteProofHigherHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	css := NewConsensusStates(t.localState, nil, nil, nil, nil, nil, nil, nil, nil)
	t.NotNil(css)

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localState.LastBlock().Height() + 3,
			round:  Round(2), // round is not important to go
		},
		previousBlock: t.localState.LastBlock().Hash(),
		previousRound: t.localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	t.NoError(css.newVoteProof(vp))

	ctx := <-css.stateChan

	t.Equal(ConsensusStateSyncing, ctx.toState)
	t.Equal(StageINIT, ctx.voteProof.Stage())
	t.Equal(initFact, ctx.voteProof.Majority())
}

func TestConsensusStates(t *testing.T) {
	suite.Run(t, new(testConsensusStates))
}
