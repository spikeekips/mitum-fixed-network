package config

import (
	"time"

	"github.com/spikeekips/mitum/base"
)

type Policy interface {
	ThresholdRatio() base.ThresholdRatio
	SetThresholdRatio(float64) error
	MaxOperationsInSeal() uint
	SetMaxOperationsInSeal(uint) error
	MaxOperationsInProposal() uint
	SetMaxOperationsInProposal(uint) error
	TimeoutWaitingProposal() time.Duration
	SetTimeoutWaitingProposal(string) error
	IntervalBroadcastingINITBallot() time.Duration
	SetIntervalBroadcastingINITBallot(string) error
	IntervalBroadcastingProposal() time.Duration
	SetIntervalBroadcastingProposal(string) error
	WaitBroadcastingACCEPTBallot() time.Duration
	SetWaitBroadcastingACCEPTBallot(string) error
	IntervalBroadcastingACCEPTBallot() time.Duration
	SetIntervalBroadcastingACCEPTBallot(string) error
	TimespanValidBallot() time.Duration
	SetTimespanValidBallot(string) error
	NetworkConnectionTimeout() time.Duration
	SetNetworkConnectionTimeout(string) error
}

type BasePolicy struct {
	thresholdRatio                   base.ThresholdRatio
	maxOperationsInSeal              uint
	maxOperationsInProposal          uint
	timeoutWaitingProposal           time.Duration
	intervalBroadcastingINITBallot   time.Duration
	intervalBroadcastingProposal     time.Duration
	waitBroadcastingACCEPTBallot     time.Duration
	intervalBroadcastingACCEPTBallot time.Duration
	timespanValidBallot              time.Duration
	networkConnectionTimeout         time.Duration
}

func (no BasePolicy) ThresholdRatio() base.ThresholdRatio {
	return no.thresholdRatio
}

func (no *BasePolicy) SetThresholdRatio(s float64) error {
	t := base.ThresholdRatio(s)
	if err := t.IsValid(nil); err != nil {
		return err
	} else {
		no.thresholdRatio = t

		return nil
	}
}

func (no *BasePolicy) MaxOperationsInSeal() uint {
	return no.maxOperationsInSeal
}

func (no *BasePolicy) SetMaxOperationsInSeal(m uint) error {
	no.maxOperationsInSeal = m

	return nil
}

func (no *BasePolicy) MaxOperationsInProposal() uint {
	return no.maxOperationsInProposal
}

func (no *BasePolicy) SetMaxOperationsInProposal(m uint) error {
	no.maxOperationsInProposal = m

	return nil
}

func (no BasePolicy) TimeoutWaitingProposal() time.Duration {
	return no.timeoutWaitingProposal
}

func (no *BasePolicy) SetTimeoutWaitingProposal(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.timeoutWaitingProposal = t

		return nil
	}
}

func (no BasePolicy) IntervalBroadcastingINITBallot() time.Duration {
	return no.intervalBroadcastingINITBallot
}

func (no *BasePolicy) SetIntervalBroadcastingINITBallot(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.intervalBroadcastingINITBallot = t

		return nil
	}
}

func (no BasePolicy) IntervalBroadcastingProposal() time.Duration {
	return no.intervalBroadcastingProposal
}

func (no *BasePolicy) SetIntervalBroadcastingProposal(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.intervalBroadcastingProposal = t

		return nil
	}
}

func (no BasePolicy) WaitBroadcastingACCEPTBallot() time.Duration {
	return no.waitBroadcastingACCEPTBallot
}

func (no *BasePolicy) SetWaitBroadcastingACCEPTBallot(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.waitBroadcastingACCEPTBallot = t

		return nil
	}
}

func (no BasePolicy) IntervalBroadcastingACCEPTBallot() time.Duration {
	return no.intervalBroadcastingACCEPTBallot
}

func (no *BasePolicy) SetIntervalBroadcastingACCEPTBallot(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.intervalBroadcastingACCEPTBallot = t

		return nil
	}
}

func (no BasePolicy) TimespanValidBallot() time.Duration {
	return no.timespanValidBallot
}

func (no *BasePolicy) SetTimespanValidBallot(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.timespanValidBallot = t

		return nil
	}
}

func (no BasePolicy) NetworkConnectionTimeout() time.Duration {
	return no.networkConnectionTimeout
}

func (no *BasePolicy) SetNetworkConnectionTimeout(s string) error {
	if t, err := parseTimeDuration(s, true); err != nil {
		return err
	} else {
		no.networkConnectionTimeout = t

		return nil
	}
}
