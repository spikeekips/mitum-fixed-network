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

type dummySyncingStateHandler struct {
	*StateSyncingHandler
}

func (ss *dummySyncingStateHandler) Activate(_ *StateChangeContext) error {
	return nil
}

func (ss *dummySyncingStateHandler) NewVoteproof(_ base.Voteproof) error {
	return nil
}

func (t *testConsensusStates) TestINITVoteproofHigherHeight() {
	ls := t.localstates(2)
	local, remote := ls[0], ls[1]

	r := base.ThresholdRatio(67)
	_ = local.Policy().SetThresholdRatio(r)
	_ = remote.Policy().SetThresholdRatio(r)

	cs := &dummySyncingStateHandler{NewStateSyncingHandler(local)}

	css, err := NewConsensusStates(local, nil, nil, nil, nil, nil, cs, nil)
	t.NoError(err)
	t.NotNil(css)
	css.activeHandler = cs

	manifest := t.lastManifest(local.Storage())
	initFact := ballot.NewINITBallotV0(
		local.Node().Address(),
		manifest.Height()+3,
		base.Round(2), // round is not important to go
		manifest.Hash(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, local, remote)
	t.NoError(err)

	t.NoError(css.newVoteproof(vp))

	t.NotNil(css.ActiveHandler())
	t.Equal(base.StateSyncing, css.ActiveHandler().State())
}

func TestConsensusStates(t *testing.T) {
	suite.Run(t, new(testConsensusStates))
}
