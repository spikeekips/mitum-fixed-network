package contestlib

import (
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/isaac"
)

type ContestPolicyDesign struct {
	isaac.PolicyOperationBodyV0
}

func NewContestPolicyDesign() *ContestPolicyDesign {
	return &ContestPolicyDesign{
		PolicyOperationBodyV0: isaac.DefaultPolicy(),
	}
}

func (cd *ContestPolicyDesign) MarshalYAML() (interface{}, error) {
	return cd.PolicyOperationBodyV0, nil
}

func (cd *ContestPolicyDesign) UnmarshalYAML(v *yaml.Node) error {
	var p isaac.PolicyOperationBodyV0
	if err := v.Decode(&p); err != nil {
		return err
	}

	d := isaac.DefaultPolicy()
	if p.ThresholdRatio() < 1 {
		p = p.SetThresholdRatio(d.ThresholdRatio())
	}
	if p.TimeoutWaitingProposal() < 1 {
		p = p.SetTimeoutWaitingProposal(d.TimeoutWaitingProposal())
	}
	if p.IntervalBroadcastingINITBallot() < 1 {
		p = p.SetIntervalBroadcastingINITBallot(d.IntervalBroadcastingINITBallot())
	}
	if p.IntervalBroadcastingProposal() < 1 {
		p = p.SetIntervalBroadcastingProposal(d.IntervalBroadcastingProposal())
	}
	if p.WaitBroadcastingACCEPTBallot() < 1 {
		p = p.SetWaitBroadcastingACCEPTBallot(d.WaitBroadcastingACCEPTBallot())
	}
	if p.IntervalBroadcastingACCEPTBallot() < 1 {
		p = p.SetIntervalBroadcastingACCEPTBallot(d.IntervalBroadcastingACCEPTBallot())
	}
	if p.NumberOfActingSuffrageNodes() < 1 {
		p = p.SetNumberOfActingSuffrageNodes(d.NumberOfActingSuffrageNodes())
	}
	if p.TimespanValidBallot() < 1 {
		p = p.SetTimespanValidBallot(d.TimespanValidBallot())
	}
	if p.TimeoutProcessProposal() < 1 {
		p = p.SetTimeoutProcessProposal(d.TimeoutProcessProposal())
	}

	cd.PolicyOperationBodyV0 = p

	return nil
}

func (cd *ContestPolicyDesign) IsValid([]byte) error {
	return cd.PolicyOperationBodyV0.IsValid(nil)
}
