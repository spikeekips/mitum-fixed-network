package config

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BasePolicyPackerJSON struct {
	ThresholdRatio                   base.ThresholdRatio `json:"threshold,omitempty"`
	MaxOperationsInSeal              uint                `json:"max_operations_in_seal"`
	MaxOperationsInProposal          uint                `json:"max_operations_in_proposal"`
	TimeoutWaitingProposal           string              `json:"timeout_waiting_proposal,omitempty"`
	IntervalBroadcastingINITBallot   string              `json:"interval_broadcasting_init_ballot,omitempty"`
	IntervalBroadcastingProposal     string              `json:"interval_broadcasting_proposal,omitempty"`
	WaitBroadcastingACCEPTBallot     string              `json:"wait_broadcasting_accept_ballot,omitempty"`
	IntervalBroadcastingACCEPTBallot string              `json:"interval_broadcasting_accept_ballot,omitempty"`
	TimespanValidBallot              string              `json:"timespan_valid_ballot,omitempty"`
	NetworkConnectionTimeout         string              `json:"network_connection_timeout,omitempty"`
}

func (no BasePolicy) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BasePolicyPackerJSON{
		ThresholdRatio:                   no.thresholdRatio,
		MaxOperationsInSeal:              no.maxOperationsInSeal,
		MaxOperationsInProposal:          no.maxOperationsInProposal,
		TimeoutWaitingProposal:           no.timeoutWaitingProposal.String(),
		IntervalBroadcastingINITBallot:   no.intervalBroadcastingINITBallot.String(),
		IntervalBroadcastingProposal:     no.intervalBroadcastingProposal.String(),
		WaitBroadcastingACCEPTBallot:     no.waitBroadcastingACCEPTBallot.String(),
		IntervalBroadcastingACCEPTBallot: no.intervalBroadcastingACCEPTBallot.String(),
		TimespanValidBallot:              no.timespanValidBallot.String(),
		NetworkConnectionTimeout:         no.networkConnectionTimeout.String(),
	})
}
