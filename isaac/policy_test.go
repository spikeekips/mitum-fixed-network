package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testPolicy struct {
	suite.Suite
	StorageSupportTest

	pk key.BTCPrivatekey
}

func (t *testPolicy) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.pk, _ = key.NewBTCPrivatekey()

	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(state.StateV0{})
	_ = t.Encs.AddHinter(state.OperationInfoV0{})
	_ = t.Encs.AddHinter(PolicyOperationBodyV0{})
	_ = t.Encs.AddHinter(SetPolicyOperationV0{})
	_ = t.Encs.AddHinter(SetPolicyOperationFactV0{})
	_ = t.Encs.AddHinter(state.HintedValue{})
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
	st := t.Storage(nil, nil)
	statepool := NewStatePool(st)

	policies := DefaultPolicy()
	policies.TimeoutWaitingProposal = policies.TimeoutWaitingProposal * 3

	spo, err := NewSetPolicyOperationV0(t.pk, []byte("this-is-token"), policies, nil)
	t.NoError(err)
	t.NoError(spo.IsValid(nil))

	previousBlock, err := block.NewTestBlockV0(
		base.Height(33),
		base.Round(2),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(err)

	currentBlock, err := block.NewTestBlockV0(
		previousBlock.Height()+1,
		base.Round(0),
		valuehash.RandomSHA256(),
		previousBlock.Hash(),
	)
	t.NoError(err)

	newState, err := spo.ProcessOperation(statepool.Get, statepool.Set)
	t.NoError(err)

	t.NoError(newState.SetPreviousBlock(previousBlock.Hash()))
	t.NoError(newState.SetCurrentBlock(currentBlock.Hash()))
	t.NoError(newState.SetHash(newState.GenerateHash()))

	t.NoError(st.NewState(newState))

	p, err := NewLocalPolicy(st, nil)
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
