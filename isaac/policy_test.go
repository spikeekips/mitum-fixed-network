package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/stretchr/testify/suite"
)

type testPolicy struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testPolicy) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testPolicy) TestLoadWithoutStorage() {
	p, err := NewLocalPolicy(nil, nil)
	t.NoError(err)

	df := DefaultPolicy()

	t.Equal(df.Threshold, p.Threshold())
	t.Equal(df.TimeoutWaitingProposal, p.TimeoutWaitingProposal())
	t.Equal(df.IntervalBroadcastingINITBallot, p.IntervalBroadcastingINITBallot())
	t.Equal(df.IntervalBroadcastingProposal, p.IntervalBroadcastingProposal())
	t.Equal(df.WaitBroadcastingACCEPTBallot, p.WaitBroadcastingACCEPTBallot())
	t.Equal(df.IntervalBroadcastingACCEPTBallot, p.IntervalBroadcastingACCEPTBallot())
	t.Equal(df.NumberOfActingSuffrageNodes, p.NumberOfActingSuffrageNodes())
	t.Equal(df.TimespanValidBallot, p.TimespanValidBallot())
}

func (t *testPolicy) TestLoadFromStorage() {
	encs := encoder.NewEncoders()
	enc := encoder.NewJSONEncoder()
	_ = encs.AddEncoder(enc)

	_ = encs.AddHinter(valuehash.SHA256{})
	_ = encs.AddHinter(state.StateV0{})
	_ = encs.AddHinter(state.OperationInfoV0{})
	_ = encs.AddHinter(PolicyOperationBodyV0{})
	_ = encs.AddHinter(SetPolicyOperationV0{})
	_ = encs.AddHinter(SetPolicyOperationFactV0{})
	_ = encs.AddHinter(state.HintedValue{})

	storage := NewMemStorage(encs, enc)
	statepool := NewStatePool(storage)

	policies := DefaultPolicy()
	policies.TimeoutWaitingProposal = policies.TimeoutWaitingProposal * 3

	spo, err := NewSetPolicyOperationV0(t.pk, []byte("this-is-token"), policies, nil)
	t.NoError(err)
	t.NoError(spo.IsValid(nil))

	newState, err := spo.ProcessOperation(statepool.Get, statepool.Set)
	t.NoError(err)

	t.NoError(storage.NewState(newState))

	p, err := NewLocalPolicy(storage, nil)
	t.NoError(err)
	t.NotNil(p)

	df := DefaultPolicy()

	t.Equal(df.Threshold, p.Threshold())

	t.NotEqual(df.TimeoutWaitingProposal, p.TimeoutWaitingProposal())
	t.Equal(policies.TimeoutWaitingProposal, p.TimeoutWaitingProposal())

	t.Equal(df.IntervalBroadcastingINITBallot, p.IntervalBroadcastingINITBallot())
	t.Equal(df.IntervalBroadcastingProposal, p.IntervalBroadcastingProposal())
	t.Equal(df.WaitBroadcastingACCEPTBallot, p.WaitBroadcastingACCEPTBallot())
	t.Equal(df.IntervalBroadcastingACCEPTBallot, p.IntervalBroadcastingACCEPTBallot())
	t.Equal(df.NumberOfActingSuffrageNodes, p.NumberOfActingSuffrageNodes())
	t.Equal(df.TimespanValidBallot, p.TimespanValidBallot())
}

func TestPolicy(t *testing.T) {
	suite.Run(t, new(testPolicy))
}
