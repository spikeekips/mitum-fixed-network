package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
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
	_ = t.Encs.AddHinter(state.HintedValue{})
	_ = t.Encs.AddHinter(policy.PolicyV0{})
	_ = t.Encs.AddHinter(policy.SetPolicyFactV0{})
	_ = t.Encs.AddHinter(policy.SetPolicyV0{})
}

func (t *testPolicy) TestLoadWithoutStorage() {
	p := NewLocalPolicy(nil)

	t.Equal(policy.DefaultPolicyThresholdRatio, p.ThresholdRatio())
	t.Equal(DefaultPolicyTimeoutWaitingProposal, p.TimeoutWaitingProposal())
	t.Equal(DefaultPolicyIntervalBroadcastingINITBallot, p.IntervalBroadcastingINITBallot())
	t.Equal(DefaultPolicyIntervalBroadcastingProposal, p.IntervalBroadcastingProposal())
	t.Equal(DefaultPolicyWaitBroadcastingACCEPTBallot, p.WaitBroadcastingACCEPTBallot())
	t.Equal(DefaultPolicyIntervalBroadcastingACCEPTBallot, p.IntervalBroadcastingACCEPTBallot())
	t.Equal(policy.DefaultPolicyNumberOfActingSuffrageNodes, p.NumberOfActingSuffrageNodes())
	t.Equal(DefaultPolicyTimespanValidBallot, p.TimespanValidBallot())
	t.Equal(DefaultPolicyTimeoutProcessProposal, p.TimeoutProcessProposal())
	t.Equal(policy.DefaultPolicyMaxOperationsInSeal, p.MaxOperationsInSeal())
	t.Equal(policy.DefaultPolicyMaxOperationsInProposal, p.MaxOperationsInProposal())
}

func (t *testPolicy) TestLoadFromStorage() {
	st := t.Storage(nil, nil)
	defer st.Close()

	statepool, err := NewStatepool(st)
	t.NoError(err)

	po := policy.NewPolicyV0(base.ThresholdRatio(99), 3, 6, policy.DefaultPolicyMaxOperationsInProposal+1)
	spo, err := policy.NewSetPolicyV0(po, util.UUID().Bytes(), t.pk, TestNetworkID)
	t.NoError(err)
	t.NoError(spo.IsValid(TestNetworkID))

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

	err = spo.Process(statepool.Get, func(op valuehash.Hash, s ...state.State) error {
		return statepool.Set(spo.Hash(), s...)
	})
	t.NoError(err)
	t.Equal(1, len(statepool.Updates()))

	var newState *state.StateUpdater
	for _, v := range statepool.Updates() {
		newState = v
		break
	}

	newState.SetHeight(currentBlock.Height())
	t.NoError(newState.SetHash(newState.GenerateHash()))

	t.NoError(st.(storage.StateUpdater).NewState(newState.GetState()))

	if mst, ok := st.(DummyMongodbStorage); ok {
		mst.SetLastBlock(currentBlock)
	}

	p := NewLocalPolicy(nil)
	t.NotNil(p)
	h, up, err := LoadLocalPolicy(st)
	t.NoError(err)
	t.NotNil(up)
	t.NotNil(h)
	t.NoError(p.Merge(up))

	t.Equal(spo.Fact().(policy.Policy).MaxOperationsInProposal(), p.MaxOperationsInProposal())
}

func TestPolicy(t *testing.T) {
	suite.Run(t, new(testPolicy))
}
