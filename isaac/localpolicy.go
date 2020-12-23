package isaac

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
)

var (
	// NOTE default threshold ratio assumes only one node exists, it means the network is just booted.
	DefaultPolicyThresholdRatio                        = base.ThresholdRatio(100)
	DefaultPolicyNumberOfActingSuffrageNodes           = uint(1)
	DefaultPolicyMaxOperationsInSeal              uint = 100
	DefaultPolicyMaxOperationsInProposal          uint = 100
	DefaultPolicyTimeoutWaitingProposal                = time.Second * 5
	DefaultPolicyIntervalBroadcastingINITBallot        = time.Second * 1
	DefaultPolicyIntervalBroadcastingProposal          = time.Second * 1
	DefaultPolicyWaitBroadcastingACCEPTBallot          = time.Second * 2
	DefaultPolicyIntervalBroadcastingACCEPTBallot      = time.Second * 1
	DefaultPolicyTimespanValidBallot                   = time.Minute * 1
	DefaultPolicyTimeoutProcessProposal                = time.Second * 30
)

type LocalPolicy struct {
	sync.RWMutex
	networkID                        *util.LockedItem
	thresholdRatio                   *util.LockedItem
	maxOperationsInSeal              *util.LockedItem
	maxOperationsInProposal          *util.LockedItem
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
		thresholdRatio:                   util.NewLockedItem(DefaultPolicyThresholdRatio),
		maxOperationsInSeal:              util.NewLockedItem(DefaultPolicyMaxOperationsInSeal),
		maxOperationsInProposal:          util.NewLockedItem(DefaultPolicyMaxOperationsInProposal),
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

func (lp *LocalPolicy) NetworkID() []byte {
	return lp.networkID.Value().([]byte)
}

func (lp *LocalPolicy) ThresholdRatio() base.ThresholdRatio {
	return lp.thresholdRatio.Value().(base.ThresholdRatio)
}

func (lp *LocalPolicy) SetThresholdRatio(ratio base.ThresholdRatio) *LocalPolicy {
	_ = lp.thresholdRatio.Set(ratio)

	return lp
}

func (lp *LocalPolicy) TimeoutWaitingProposal() time.Duration {
	return lp.timeoutWaitingProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimeoutWaitingProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimeoutWaitingProposal too short; %v", d)
	}

	_ = lp.timeoutWaitingProposal.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingINITBallot() time.Duration {
	return lp.intervalBroadcastingINITBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingINITBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingINITBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingINITBallot.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingProposal() time.Duration {
	return lp.intervalBroadcastingProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingProposal too short; %v", d)
	}

	_ = lp.intervalBroadcastingProposal.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) WaitBroadcastingACCEPTBallot() time.Duration {
	return lp.waitBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetWaitBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("WaitBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.waitBroadcastingACCEPTBallot.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) IntervalBroadcastingACCEPTBallot() time.Duration {
	return lp.intervalBroadcastingACCEPTBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetIntervalBroadcastingACCEPTBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("IntervalBroadcastingACCEPTBallot too short; %v", d)
	}

	_ = lp.intervalBroadcastingACCEPTBallot.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) TimespanValidBallot() time.Duration {
	return lp.timespanValidBallot.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimespanValidBallot(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimespanValidBallot too short; %v", d)
	}

	_ = lp.timespanValidBallot.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) TimeoutProcessProposal() time.Duration {
	return lp.timeoutProcessProposal.Value().(time.Duration)
}

func (lp *LocalPolicy) SetTimeoutProcessProposal(d time.Duration) (*LocalPolicy, error) {
	if d < 1 {
		return nil, xerrors.Errorf("TimeoutProcessProposal too short; %v", d)
	}

	_ = lp.timeoutProcessProposal.Set(d)

	return lp, nil
}

func (lp *LocalPolicy) MaxOperationsInSeal() uint {
	return lp.maxOperationsInSeal.Value().(uint)
}

func (lp *LocalPolicy) SetMaxOperationsInSeal(m uint) (*LocalPolicy, error) {
	if m < 1 {
		return nil, xerrors.Errorf("zero MaxOperationsInSeal")
	}

	_ = lp.maxOperationsInSeal.Set(m)

	return lp, nil
}

func (lp *LocalPolicy) MaxOperationsInProposal() uint {
	return lp.maxOperationsInProposal.Value().(uint)
}

func (lp *LocalPolicy) SetMaxOperationsInProposal(m uint) (*LocalPolicy, error) {
	if m < 1 {
		return nil, xerrors.Errorf("zero MaxOperationsInProposal")
	}

	_ = lp.maxOperationsInProposal.Set(m)

	return lp, nil
}

func (lp *LocalPolicy) Config() map[string]interface{} {
	return map[string]interface{}{
		"threshold":                           lp.ThresholdRatio(),
		"max_operations_in_seal":              lp.MaxOperationsInSeal(),
		"max_operations_in_proposal":          lp.MaxOperationsInProposal(),
		"timeout_waiting_proposal":            lp.TimeoutWaitingProposal(),
		"interval_broadcasting_init_ballot":   lp.IntervalBroadcastingINITBallot(),
		"interval_broadcasting_proposal":      lp.IntervalBroadcastingProposal(),
		"wait_broadcasting_accept_ballot":     lp.WaitBroadcastingACCEPTBallot(),
		"interval_broadcasting_accept_ballot": lp.IntervalBroadcastingACCEPTBallot(),
		"timespan_valid_ballot":               lp.TimespanValidBallot(),
		"timeout_process_proposal":            lp.TimeoutProcessProposal(),
	}
}
