package config

import (
	"time"

	"github.com/spikeekips/mitum/base"
)

type BasePolicyPackerYAML struct {
	ThresholdRatio                   base.ThresholdRatio `yaml:"threshold,omitempty"`
	MaxOperationsInSeal              uint                `yaml:"max-operations-in-seal"`
	MaxOperationsInProposal          uint                `yaml:"max-operations-in-proposal"`
	TimeoutWaitingProposal           time.Duration       `yaml:"timeout-waiting-proposal,omitempty"`
	IntervalBroadcastingINITBallot   time.Duration       `yaml:"interval-broadcasting-init-ballot,omitempty"`
	IntervalBroadcastingProposal     time.Duration       `yaml:"interval-broadcasting-proposal,omitempty"`
	WaitBroadcastingACCEPTBallot     time.Duration       `yaml:"wait-broadcasting-accept-ballot,omitempty"`
	IntervalBroadcastingACCEPTBallot time.Duration       `yaml:"interval-broadcasting-accept-ballot,omitempty"`
	TimespanValidBallot              time.Duration       `yaml:"timespan-valid-ballot,omitempty"`
	TimeoutProcessProposal           time.Duration       `yaml:"timeout-process-proposal,omitempty"`
	NetworkConnectionTimeout         time.Duration       `yaml:"network-connection-timeout,omitempty"`
	NetworkConnectionTLSInsecure     bool                `yaml:"network-connection-tls-insecure,omitempty"`
}

func (no BasePolicy) MarshalYAML() (interface{}, error) {
	return BasePolicyPackerYAML{
		ThresholdRatio:                   no.thresholdRatio,
		MaxOperationsInSeal:              no.maxOperationsInSeal,
		MaxOperationsInProposal:          no.maxOperationsInProposal,
		TimeoutWaitingProposal:           no.timeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   no.intervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     no.intervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     no.waitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: no.intervalBroadcastingACCEPTBallot,
		TimespanValidBallot:              no.timespanValidBallot,
		TimeoutProcessProposal:           no.timeoutProcessProposal,
		NetworkConnectionTimeout:         no.networkConnectionTimeout,
		NetworkConnectionTLSInsecure:     no.networkConnectionTLSInsecure,
	}, nil
}
