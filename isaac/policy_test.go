package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testPolicy struct {
	suite.Suite
	StorageSupportTest

	pk key.Privatekey
}

func (t *testPolicy) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.pk, _ = key.NewBTCPrivatekey()

	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(state.StateV0{})
	_ = t.Encs.AddHinter(operation.OperationInfoV0{})
	_ = t.Encs.AddHinter(PolicyOperationBodyV0{})
	_ = t.Encs.AddHinter(SetPolicyOperationV0{})
	_ = t.Encs.AddHinter(SetPolicyOperationFactV0{})
	_ = t.Encs.AddHinter(state.HintedValue{})
}

func (t *testPolicy) TestLoadWithoutStorage() {
	p, err := NewLocalPolicy(nil, nil)
	t.NoError(err)

	df := DefaultPolicy()

	t.Equal(df.ThresholdRatio(), p.ThresholdRatio())
	t.Equal(df.TimeoutWaitingProposal(), p.TimeoutWaitingProposal())
	t.Equal(df.IntervalBroadcastingINITBallot(), p.IntervalBroadcastingINITBallot())
	t.Equal(df.IntervalBroadcastingProposal(), p.IntervalBroadcastingProposal())
	t.Equal(df.WaitBroadcastingACCEPTBallot(), p.WaitBroadcastingACCEPTBallot())
	t.Equal(df.IntervalBroadcastingACCEPTBallot(), p.IntervalBroadcastingACCEPTBallot())
	t.Equal(df.NumberOfActingSuffrageNodes(), p.NumberOfActingSuffrageNodes())
	t.Equal(df.TimespanValidBallot(), p.TimespanValidBallot())
	t.Equal(df.TimeoutProcessProposal(), p.TimeoutProcessProposal())
}

func (t *testPolicy) TestLoadFromStorage() {
	st := t.Storage(nil, nil)
	defer st.Close()

	statepool, err := NewStatepool(st)
	t.NoError(err)

	policies := DefaultPolicy()
	policies.timeoutWaitingProposal = policies.timeoutWaitingProposal * 3

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

	var newStates []state.StateUpdater
	err = spo.Process(statepool.Get, func(op valuehash.Hash, s ...state.StateUpdater) error {
		newStates = s
		return statepool.Set(spo.Hash(), s...)
	})
	t.NoError(err)
	t.Equal(1, len(newStates))

	newState := newStates[0]
	t.NoError(newState.SetPreviousBlock(previousBlock.Hash()))
	t.NoError(newState.SetCurrentBlock(currentBlock.Height(), currentBlock.Hash()))
	t.NoError(newState.SetHash(newState.GenerateHash()))

	t.NoError(st.NewState(newState))

	if mst, ok := st.(DummyMongodbStorage); ok {
		mst.SetLastBlock(currentBlock)
	}

	p, err := NewLocalPolicy(st, nil)
	t.NoError(err)
	t.NotNil(p)

	df := DefaultPolicy()

	t.Equal(df.thresholdRatio, p.ThresholdRatio())

	t.NotEqual(df.timeoutWaitingProposal, p.TimeoutWaitingProposal())
	t.Equal(policies.timeoutWaitingProposal, p.TimeoutWaitingProposal())

	t.Equal(df.intervalBroadcastingINITBallot, p.IntervalBroadcastingINITBallot())
	t.Equal(df.intervalBroadcastingProposal, p.IntervalBroadcastingProposal())
	t.Equal(df.waitBroadcastingACCEPTBallot, p.WaitBroadcastingACCEPTBallot())
	t.Equal(df.intervalBroadcastingACCEPTBallot, p.IntervalBroadcastingACCEPTBallot())
	t.Equal(df.numberOfActingSuffrageNodes, p.NumberOfActingSuffrageNodes())
	t.Equal(df.timespanValidBallot, p.TimespanValidBallot())
	t.Equal(df.timeoutProcessProposal, p.TimeoutProcessProposal())
}

func TestPolicy(t *testing.T) {
	suite.Run(t, new(testPolicy))
}
