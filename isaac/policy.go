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
	thresholdRatio                   *util.LockedItem
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

	d := DefaultPolicy()
	lp.thresholdRatio = util.NewLockedItem(d.ThresholdRatio())
	lp.timeoutWaitingProposal = util.NewLockedItem(d.TimeoutWaitingProposal())
	lp.intervalBroadcastingINITBallot = util.NewLockedItem(d.IntervalBroadcastingINITBallot())
	lp.intervalBroadcastingProposal = util.NewLockedItem(d.IntervalBroadcastingProposal())
	lp.waitBroadcastingACCEPTBallot = util.NewLockedItem(d.WaitBroadcastingACCEPTBallot())
	lp.intervalBroadcastingACCEPTBallot = util.NewLockedItem(d.IntervalBroadcastingACCEPTBallot())
	lp.numberOfActingSuffrageNodes = util.NewLockedItem(d.NumberOfActingSuffrageNodes())
	lp.timespanValidBallot = util.NewLockedItem(d.TimespanValidBallot())
	lp.timeoutProcessProposal = util.NewLockedItem(d.TimeoutProcessProposal())

	if err := lp.load(); err != nil {
		return nil, err
	}

	return lp, nil
}

func (lp *LocalPolicy) load() error {
	var loaded base.PolicyOperationBody
	if lp.st == nil {
		loaded = DefaultPolicy()
	} else {
		if l, found, err := lp.st.State(PolicyOperationKey); err != nil {
			return err
		} else if !found || l.Value() == nil { // set default
			loaded = DefaultPolicy()
		} else if i := l.Value().Interface(); i == nil {
			loaded = DefaultPolicy()
		} else if p, ok := i.(base.PolicyOperationBody); !ok {
			loaded = DefaultPolicy()
		} else {
			loaded = p
		}
	}

	return lp.Merge(loaded)
}

func (lp *LocalPolicy) Reload() error {
	lp.Lock()
	defer lp.Unlock()

	return lp.load()
}

func (lp *LocalPolicy) Merge(p base.PolicyOperationBody) error {
	if v := lp.ThresholdRatio(); v != p.ThresholdRatio() {
		_ = lp.thresholdRatio.SetValue(p.ThresholdRatio())
	}
	if v := lp.TimeoutWaitingProposal(); v != p.TimeoutWaitingProposal() {
		_ = lp.timeoutWaitingProposal.SetValue(p.TimeoutWaitingProposal())
	}
	if v := lp.IntervalBroadcastingINITBallot(); v != p.IntervalBroadcastingINITBallot() {
		_ = lp.intervalBroadcastingINITBallot.SetValue(p.IntervalBroadcastingINITBallot())
	}
	if v := lp.IntervalBroadcastingProposal(); v != p.IntervalBroadcastingProposal() {
		_ = lp.intervalBroadcastingProposal.SetValue(p.IntervalBroadcastingProposal())
	}
	if v := lp.WaitBroadcastingACCEPTBallot(); v != p.WaitBroadcastingACCEPTBallot() {
		_ = lp.waitBroadcastingACCEPTBallot.SetValue(p.WaitBroadcastingACCEPTBallot())
	}
	if v := lp.IntervalBroadcastingACCEPTBallot(); v != p.IntervalBroadcastingACCEPTBallot() {
		_ = lp.intervalBroadcastingACCEPTBallot.SetValue(p.IntervalBroadcastingACCEPTBallot())
	}
	if v := lp.NumberOfActingSuffrageNodes(); v != p.NumberOfActingSuffrageNodes() {
		_ = lp.numberOfActingSuffrageNodes.SetValue(p.NumberOfActingSuffrageNodes())
	}
	if v := lp.TimespanValidBallot(); v != p.TimespanValidBallot() {
		_ = lp.timespanValidBallot.SetValue(p.TimespanValidBallot())
	}
	if v := lp.TimeoutProcessProposal(); v != p.TimeoutProcessProposal() {
		_ = lp.timeoutProcessProposal.SetValue(p.TimeoutProcessProposal())
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

func (lp *LocalPolicy) PolicyOperationBody() base.PolicyOperationBody {
	return PolicyOperationBodyV0{
		thresholdRatio:                   lp.ThresholdRatio(),
		timeoutWaitingProposal:           lp.TimeoutWaitingProposal(),
		intervalBroadcastingINITBallot:   lp.IntervalBroadcastingINITBallot(),
		intervalBroadcastingProposal:     lp.IntervalBroadcastingProposal(),
		waitBroadcastingACCEPTBallot:     lp.WaitBroadcastingACCEPTBallot(),
		intervalBroadcastingACCEPTBallot: lp.IntervalBroadcastingACCEPTBallot(),
		numberOfActingSuffrageNodes:      lp.NumberOfActingSuffrageNodes(),
		timespanValidBallot:              lp.TimespanValidBallot(),
		timeoutProcessProposal:           lp.TimeoutProcessProposal(),
	}
}
