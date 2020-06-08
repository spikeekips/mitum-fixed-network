package isaac

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type LocalPolicy struct {
	sync.RWMutex
	st                               storage.Storage
	networkID                        *util.LockedItem // NOTE networkID should be string, internally []byte
	threshold                        *util.LockedItem
	timeoutWaitingProposal           *util.LockedItem
	intervalBroadcastingINITBallot   *util.LockedItem
	intervalBroadcastingProposal     *util.LockedItem
	waitBroadcastingACCEPTBallot     *util.LockedItem
	intervalBroadcastingACCEPTBallot *util.LockedItem
	numberOfActingSuffrageNodes      *util.LockedItem
	// timespanValidBallot is used to check the SignedAt time of incoming
	// Ballot should be within timespanValidBallot on now. By default, 1 minute.
	timespanValidBallot    *util.LockedItem
	timeoutProcessProposal *util.LockedItem
}

func NewLocalPolicy(st storage.Storage, networkID []byte) (*LocalPolicy, error) {
	lp := &LocalPolicy{
		st:        st,
		networkID: util.NewLockedItem(networkID),
	}

	lp.threshold = util.NewLockedItem(nil)
	lp.timeoutWaitingProposal = util.NewLockedItem(nil)
	lp.intervalBroadcastingINITBallot = util.NewLockedItem(nil)
	lp.intervalBroadcastingProposal = util.NewLockedItem(nil)
	lp.waitBroadcastingACCEPTBallot = util.NewLockedItem(nil)
	lp.intervalBroadcastingACCEPTBallot = util.NewLockedItem(nil)
	lp.numberOfActingSuffrageNodes = util.NewLockedItem(nil)
	lp.timespanValidBallot = util.NewLockedItem(nil)
	lp.timeoutProcessProposal = util.NewLockedItem(nil)

	if err := lp.load(); err != nil {
		return nil, err
	}

	return lp, nil
}

func (lp *LocalPolicy) load() error {
	var loaded PolicyOperationBodyV0
	if lp.st == nil {
		loaded = DefaultPolicy()
	} else {
		if l, found, err := lp.st.State(PolicyOperationKey); err != nil {
			return err
		} else if !found || l.Value() == nil { // set default
			loaded = DefaultPolicy()
		} else if i := l.Value().Interface(); i == nil {
			loaded = DefaultPolicy()
		} else if p, ok := i.(PolicyOperationBodyV0); !ok {
			loaded = DefaultPolicy()
		} else {
			loaded = p
		}
	}

	lp.threshold.SetValue(loaded.Threshold)
	lp.timeoutWaitingProposal.SetValue(loaded.TimeoutWaitingProposal)
	lp.intervalBroadcastingINITBallot.SetValue(loaded.IntervalBroadcastingINITBallot)
	lp.intervalBroadcastingProposal.SetValue(loaded.IntervalBroadcastingProposal)
	lp.waitBroadcastingACCEPTBallot.SetValue(loaded.WaitBroadcastingACCEPTBallot)
	lp.intervalBroadcastingACCEPTBallot.SetValue(loaded.IntervalBroadcastingACCEPTBallot)
	lp.numberOfActingSuffrageNodes.SetValue(loaded.NumberOfActingSuffrageNodes)
	lp.timespanValidBallot.SetValue(loaded.TimespanValidBallot)
	lp.timeoutProcessProposal.SetValue(loaded.TimeoutProcessProposal)

	return nil
}

func (lp *LocalPolicy) Reload() error {
	lp.Lock()
	defer lp.Unlock()

	return lp.load()
}

func (lp *LocalPolicy) NetworkID() []byte {
	return lp.networkID.Value().([]byte)
}

func (lp *LocalPolicy) Threshold() base.Threshold {
	return lp.threshold.Value().(base.Threshold)
}

func (lp *LocalPolicy) SetThreshold(threshold base.Threshold) *LocalPolicy {
	_ = lp.threshold.SetValue(threshold)

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
