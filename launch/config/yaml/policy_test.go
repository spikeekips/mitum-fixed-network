package yamlconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type testPolicy struct {
	suite.Suite
}

func (t *testPolicy) TestEmpty() {
	y := ""

	var n Policy
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Nil(n.ThresholdRatio)
	t.Nil(n.TimeoutWaitingProposal)
	t.Nil(n.IntervalBroadcastingINITBallot)
	t.Nil(n.IntervalBroadcastingProposal)
	t.Nil(n.WaitBroadcastingACCEPTBallot)
	t.Nil(n.IntervalBroadcastingACCEPTBallot)
	t.Nil(n.TimespanValidBallot)
	t.Nil(n.TimeoutProcessProposal)
}

func (t *testPolicy) TestThresholdRatio() {
	{
		y := `
threshold: 33
`

		var n Policy
		err := yaml.Unmarshal([]byte(y), &n)
		t.NoError(err)

		t.Nil(n.TimeoutWaitingProposal)
		t.Nil(n.IntervalBroadcastingINITBallot)
		t.Nil(n.IntervalBroadcastingProposal)
		t.Nil(n.WaitBroadcastingACCEPTBallot)
		t.Nil(n.IntervalBroadcastingACCEPTBallot)
		t.Nil(n.TimespanValidBallot)
		t.Nil(n.TimeoutProcessProposal)

		t.Equal(float64(33), *n.ThresholdRatio)
	}

	{
		y := `
threshold: 33.9
`

		var n Policy
		err := yaml.Unmarshal([]byte(y), &n)
		t.NoError(err)

		t.Equal(float64(33.9), *n.ThresholdRatio)
	}
}

func (t *testPolicy) TestDuration() {
	y := `
timeout-waiting-proposal: 33m3s
`

	var n Policy
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Nil(n.ThresholdRatio)
	t.Nil(n.IntervalBroadcastingINITBallot)
	t.Nil(n.IntervalBroadcastingProposal)
	t.Nil(n.WaitBroadcastingACCEPTBallot)
	t.Nil(n.IntervalBroadcastingACCEPTBallot)
	t.Nil(n.TimespanValidBallot)
	t.Nil(n.TimeoutProcessProposal)

	t.Equal("33m3s", *n.TimeoutWaitingProposal)
}

func TestPolicy(t *testing.T) {
	suite.Run(t, new(testPolicy))
}
