package config

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BasePolicyPackerJSON struct {
	ThresholdRatio                   base.ThresholdRatio `json:"threshold,omitempty"`
	TimeoutWaitingProposal           string              `json:"timeout-waiting-proposal,omitempty"`
	IntervalBroadcastingINITBallot   string              `json:"interval-broadcasting-init-ballot,omitempty"`
	IntervalBroadcastingProposal     string              `json:"interval-broadcasting-proposal,omitempty"`
	WaitBroadcastingACCEPTBallot     string              `json:"wait-broadcasting-accept-ballot,omitempty"`
	IntervalBroadcastingACCEPTBallot string              `json:"interval-broadcasting-accept-ballot,omitempty"`
	TimespanValidBallot              string              `json:"timespan-valid-ballot,omitempty"`
	TimeoutProcessProposal           string              `json:"timeout-process-proposal,omitempty"`
}

func (no BasePolicy) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BasePolicyPackerJSON{
		ThresholdRatio:                   no.thresholdRatio,
		TimeoutWaitingProposal:           no.timeoutWaitingProposal.String(),
		IntervalBroadcastingINITBallot:   no.intervalBroadcastingINITBallot.String(),
		IntervalBroadcastingProposal:     no.intervalBroadcastingProposal.String(),
		WaitBroadcastingACCEPTBallot:     no.waitBroadcastingACCEPTBallot.String(),
		IntervalBroadcastingACCEPTBallot: no.intervalBroadcastingACCEPTBallot.String(),
		TimespanValidBallot:              no.timespanValidBallot.String(),
		TimeoutProcessProposal:           no.timeoutProcessProposal.String(),
	})
}
