package isaac

import (
	"time"

	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
)

type PolicyOperationBodyV0PackerYAML struct {
	ThresholdRatio                   base.ThresholdRatio `yaml:"threshold"`
	TimeoutWaitingProposal           time.Duration       `yaml:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration       `yaml:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration       `yaml:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration       `yaml:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration       `yaml:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint                `yaml:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration       `yaml:"timespan_valid_ballot"`
	TimeoutProcessProposal           time.Duration       `yaml:"timeout_process_proposal"`
}

func (po PolicyOperationBodyV0) MarshalYAML() (interface{}, error) {
	return PolicyOperationBodyV0PackerYAML{
		ThresholdRatio:                   po.thresholdRatio,
		TimeoutWaitingProposal:           po.timeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   po.intervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     po.intervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     po.waitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: po.intervalBroadcastingACCEPTBallot,
		NumberOfActingSuffrageNodes:      po.numberOfActingSuffrageNodes,
		TimespanValidBallot:              po.timespanValidBallot,
		TimeoutProcessProposal:           po.timeoutProcessProposal,
	}, nil
}

func (po *PolicyOperationBodyV0) UnmarshalYAML(v *yaml.Node) error {
	var up PolicyOperationBodyV0PackerYAML
	if err := v.Decode(&up); err != nil {
		return err
	}

	return po.unpack(
		up.ThresholdRatio,
		up.TimeoutWaitingProposal,
		up.IntervalBroadcastingINITBallot,
		up.IntervalBroadcastingProposal,
		up.WaitBroadcastingACCEPTBallot,
		up.IntervalBroadcastingACCEPTBallot,
		up.NumberOfActingSuffrageNodes,
		up.TimespanValidBallot,
		up.TimeoutProcessProposal,
	)
}
