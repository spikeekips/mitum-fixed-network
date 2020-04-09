package isaac

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type LocalPolicy struct {
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
	timespanValidBallot *util.LockedItem
}

func NewLocalPolicy(storage Storage, networkID []byte) (*LocalPolicy, error) {
	var loaded PolicyOperationBodyV0
	if storage == nil {
		loaded = DefaultPolicy()
	} else {
		if l, found, err := storage.State(PolicyOperationKey); err != nil {
			return nil, err
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

	return &LocalPolicy{
		networkID:                        util.NewLockedItem(networkID),
		threshold:                        util.NewLockedItem(loaded.Threshold),
		timeoutWaitingProposal:           util.NewLockedItem(loaded.TimeoutWaitingProposal),
		intervalBroadcastingINITBallot:   util.NewLockedItem(loaded.IntervalBroadcastingINITBallot),
		intervalBroadcastingProposal:     util.NewLockedItem(loaded.IntervalBroadcastingProposal),
		waitBroadcastingACCEPTBallot:     util.NewLockedItem(loaded.WaitBroadcastingACCEPTBallot),
		intervalBroadcastingACCEPTBallot: util.NewLockedItem(loaded.IntervalBroadcastingACCEPTBallot),
		numberOfActingSuffrageNodes:      util.NewLockedItem(loaded.NumberOfActingSuffrageNodes),
		timespanValidBallot:              util.NewLockedItem(loaded.TimespanValidBallot),
	}, nil
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
