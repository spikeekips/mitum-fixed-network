package isaac

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	// NOTE default threshold ratio assumes only one node exists, it means the network is just booted.
	DefaultPolicyTimeoutWaitingProposal           = time.Second * 5
	DefaultPolicyIntervalBroadcastingINITBallot   = time.Second * 1
	DefaultPolicyIntervalBroadcastingProposal     = time.Second * 1
	DefaultPolicyWaitBroadcastingACCEPTBallot     = time.Second * 2
	DefaultPolicyIntervalBroadcastingACCEPTBallot = time.Second * 1
	DefaultPolicyTimespanValidBallot              = time.Minute * 1
	DefaultPolicyTimeoutProcessProposal           = time.Second * 30
)

type LocalPolicy struct {
	sync.RWMutex
	lastPolicy                       valuehash.Hash
	networkID                        *util.LockedItem
	thresholdRatio                   *util.LockedItem
	maxOperationsInSeal              *util.LockedItem
	maxOperationsInProposal          *util.LockedItem
	numberOfActingSuffrageNodes      *util.LockedItem
	timeoutWaitingProposal           *util.LockedItem
	intervalBroadcastingINITBallot   *util.LockedItem
	intervalBroadcastingProposal     *util.LockedItem
	waitBroadcastingACCEPTBallot     *util.LockedItem
	intervalBroadcastingACCEPTBallot *util.LockedItem
	// timespanValidBallot is used to check the SignedAt time of incoming
	// Ballot should be within timespanValidBallot on now. By default, 1 minute.
	timespanValidBallot    *util.LockedItem
	timeoutProcessProposal *util.LockedItem
}

func NewLocalPolicy(networkID []byte) *LocalPolicy {
	lp := &LocalPolicy{
		networkID:                        util.NewLockedItem(networkID),
		thresholdRatio:                   util.NewLockedItem(policy.DefaultPolicyThresholdRatio),
		numberOfActingSuffrageNodes:      util.NewLockedItem(policy.DefaultPolicyNumberOfActingSuffrageNodes),
		maxOperationsInSeal:              util.NewLockedItem(policy.DefaultPolicyMaxOperationsInSeal),
		maxOperationsInProposal:          util.NewLockedItem(policy.DefaultPolicyMaxOperationsInProposal),
		timeoutWaitingProposal:           util.NewLockedItem(DefaultPolicyTimeoutWaitingProposal),
		intervalBroadcastingINITBallot:   util.NewLockedItem(DefaultPolicyIntervalBroadcastingINITBallot),
		intervalBroadcastingProposal:     util.NewLockedItem(DefaultPolicyIntervalBroadcastingProposal),
		waitBroadcastingACCEPTBallot:     util.NewLockedItem(DefaultPolicyWaitBroadcastingACCEPTBallot),
		intervalBroadcastingACCEPTBallot: util.NewLockedItem(DefaultPolicyIntervalBroadcastingACCEPTBallot),
		timespanValidBallot:              util.NewLockedItem(DefaultPolicyTimespanValidBallot),
		timeoutProcessProposal:           util.NewLockedItem(DefaultPolicyTimeoutProcessProposal),
	}

	return lp
}

func LoadLocalPolicy(st storage.Storage) (valuehash.Hash, policy.Policy, error) {
	if l, found, err := st.State(policy.PolicyOperationKey); err != nil {
		return nil, nil, err
	} else if !found || l.Value() == nil { // set default
		return nil, nil, nil
	} else if i := l.Value().Interface(); i == nil {
		return nil, nil, nil
	} else if p, ok := i.(policy.Policy); !ok {
		return nil, nil, xerrors.Errorf("wrong type policy, %T", i)
	} else {
		return l.Hash(), p, nil
	}
}

func (lp *LocalPolicy) Reload(st storage.Storage) error {
	lp.Lock()
	defer lp.Unlock()

	switch h, p, err := LoadLocalPolicy(st); {
	case err != nil:
		return err
	case p == nil:
		return nil
	case lp.lastPolicy != nil && lp.lastPolicy.Equal(h):
		return nil
	default:
		if err := lp.Merge(p); err != nil {
			return err
		} else {
			lp.lastPolicy = h
			return nil
		}
	}
}

func (lp *LocalPolicy) Merge(p policy.Policy) error {
	if v := lp.ThresholdRatio(); v != p.ThresholdRatio() {
		_ = lp.thresholdRatio.SetValue(p.ThresholdRatio())
	}
	if v := lp.NumberOfActingSuffrageNodes(); v != p.NumberOfActingSuffrageNodes() {
		_ = lp.numberOfActingSuffrageNodes.SetValue(p.NumberOfActingSuffrageNodes())
	}
	if v := lp.MaxOperationsInSeal(); v != p.MaxOperationsInSeal() {
		_ = lp.maxOperationsInSeal.SetValue(p.MaxOperationsInSeal())
	}
	if v := lp.MaxOperationsInProposal(); v != p.MaxOperationsInProposal() {
		_ = lp.maxOperationsInProposal.SetValue(p.MaxOperationsInProposal())
	}

	return nil
}

func (lp *LocalPolicy) NetworkID() []byte {
	return lp.networkID.Value().([]byte)
}

func (lp *LocalPolicy) ThresholdRatio() base.ThresholdRatio {
	return lp.thresholdRatio.Value().(base.ThresholdRatio)
}

func (lp *LocalPolicy) SetThresholdRatio(ratio base.ThresholdRatio) *LocalPolicy {
	_ = lp.thresholdRatio.SetValue(ratio)

	return lp
}

func (lp *LocalPolicy) TimeoutWaitingProposal() time.Duration {
	return lp.timeoutWaitingProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimeoutWaitingProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimeoutWaitingProposal too short; %v", d)
	}

	_ = lp.timeoutWaitingProposal.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingINITBallot() time.Duration {
	return lp.intervalBroadcastingINITBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingINITBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingINITBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingINITBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingProposal() time.Duration {
	return lp.intervalBroadcastingProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingProposal too short; %v", d)
	}

	_ = lp.intervalBroadcastingProposal.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) WaitBroadcastingACCEPTBallot() time.Duration {
	return lp.waitBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetWaitBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("WaitBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.waitBroadcastingACCEPTBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingACCEPTBallot() time.Duration {
	return lp.intervalBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingACCEPTBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) NumberOfActingSuffrageNodes() uint {
	return lp.numberOfActingSuffrageNodes.Value().(uint)
}

func (lp *LocalPolicy) SetNumberOfActingSuffrageNodes(i uint) (*LocalPolicy, error) {
	if i < 1 {
		return nil, xerrors.Errorf("NumberOfActingSuffrageNodes should be greater than 0; %v", i)
	}

	_ = lp.numberOfActingSuffrageNodes.SetValue(i)

	return lp, nil
}

func (lp *LocalPolicy) TimespanValidBallot() time.Duration {
	return lp.timespanValidBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimespanValidBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimespanValidBallot too short; %v", d)
	}

	_ = lp.timespanValidBallot.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) TimeoutProcessProposal() time.Duration {
	return lp.timeoutProcessProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimeoutProcessProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimeoutProcessProposal too short; %v", d)
	}

	_ = lp.timeoutProcessProposal.SetValue(d)

	return lp, nil
}

func (lp *LocalPolicy) MaxOperationsInSeal() uint {
	return lp.maxOperationsInSeal.Value().(uint)
}

func (lp *LocalPolicy) MaxOperationsInProposal() uint {
	return lp.maxOperationsInProposal.Value().(uint)
}

func (lp *LocalPolicy) Policy() policy.Policy {
	return policy.NewPolicyV0(
		lp.ThresholdRatio(),
		lp.NumberOfActingSuffrageNodes(),
		lp.MaxOperationsInSeal(),
		lp.MaxOperationsInProposal(),
	)
}
