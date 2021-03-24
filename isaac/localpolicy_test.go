package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
)

type testPolicy struct {
	suite.Suite
}

func (t *testPolicy) TestNew() {
	p := NewLocalPolicy(nil)

	t.Equal(DefaultPolicyThresholdRatio, p.ThresholdRatio())
	t.Equal(DefaultPolicyTimeoutWaitingProposal, p.TimeoutWaitingProposal())
	t.Equal(DefaultPolicyIntervalBroadcastingINITBallot, p.IntervalBroadcastingINITBallot())
	t.Equal(DefaultPolicyIntervalBroadcastingProposal, p.IntervalBroadcastingProposal())
	t.Equal(DefaultPolicyWaitBroadcastingACCEPTBallot, p.WaitBroadcastingACCEPTBallot())
	t.Equal(DefaultPolicyIntervalBroadcastingACCEPTBallot, p.IntervalBroadcastingACCEPTBallot())
	t.Equal(DefaultPolicyTimespanValidBallot, p.TimespanValidBallot())
	t.Equal(DefaultPolicyTimeoutProcessProposal, p.TimeoutProcessProposal())
	t.Equal(DefaultPolicyMaxOperationsInSeal, p.MaxOperationsInSeal())
	t.Equal(DefaultPolicyMaxOperationsInProposal, p.MaxOperationsInProposal())
	t.Equal(DefaultPolicyNetworkConnectionTimeout, p.NetworkConnectionTimeout())
	t.Equal(DefaultPolicyNetworkConnectionTLSInsecure, p.NetworkConnectionTLSInsecure())
}

func (t *testPolicy) TestSet() {
	p := NewLocalPolicy(nil)

	th := base.ThresholdRatio(66.6)
	_ = p.SetThresholdRatio(th)
	t.Equal(th, p.ThresholdRatio())

	maxOperationsInSeal := uint(33)
	_, err := p.SetMaxOperationsInSeal(maxOperationsInSeal)
	t.NoError(err)

	t.Equal(maxOperationsInSeal, p.MaxOperationsInSeal())

	maxOperationsInProposal := uint(44)
	_, err = p.SetMaxOperationsInProposal(maxOperationsInProposal)
	t.NoError(err)

	t.Equal(maxOperationsInProposal, p.MaxOperationsInProposal())

	_, err = p.SetNetworkConnectionTimeout(time.Millisecond * 300)
	t.Contains(err.Error(), "too short")

	_, err = p.SetNetworkConnectionTimeout(time.Second)
	t.NoError(err)
	t.Equal(time.Second, p.NetworkConnectionTimeout())

	_, err = p.SetNetworkConnectionTLSInsecure(true)
	t.NoError(err)
	t.True(p.NetworkConnectionTLSInsecure())
}

func TestPolicy(t *testing.T) {
	suite.Run(t, new(testPolicy))
}
