package contestlib

import (
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ContestPolicyDesign struct {
	isaac.PolicyOperationBodyV0
	Threshold float64
}

func NewContestPolicyDesign() *ContestPolicyDesign {
	return &ContestPolicyDesign{
		PolicyOperationBodyV0: isaac.DefaultPolicy(),
		Threshold:             67,
	}
}

func (cd *ContestPolicyDesign) UnmarshalYAML(v *yaml.Node) error {
	var t struct {
		Threshold float64
	}

	if err := v.Decode(&t); err != nil {
		return err
	}
	cd.Threshold = t.Threshold

	var p isaac.PolicyOperationBodyV0
	if err := v.Decode(&p); err != nil {
		return err
	}

	d := isaac.DefaultPolicy()
	if p.TimeoutWaitingProposal < 1 {
		p.TimeoutWaitingProposal = d.TimeoutWaitingProposal
	}
	if p.IntervalBroadcastingINITBallot < 1 {
		p.IntervalBroadcastingINITBallot = d.IntervalBroadcastingINITBallot
	}
	if p.IntervalBroadcastingProposal < 1 {
		p.IntervalBroadcastingProposal = d.IntervalBroadcastingProposal
	}
	if p.WaitBroadcastingACCEPTBallot < 1 {
		p.WaitBroadcastingACCEPTBallot = d.WaitBroadcastingACCEPTBallot
	}
	if p.IntervalBroadcastingACCEPTBallot < 1 {
		p.IntervalBroadcastingACCEPTBallot = d.IntervalBroadcastingACCEPTBallot
	}
	if p.NumberOfActingSuffrageNodes < 1 {
		p.NumberOfActingSuffrageNodes = d.NumberOfActingSuffrageNodes
	}
	if p.TimespanValidBallot < 1 {
		p.TimespanValidBallot = d.TimespanValidBallot
	}
	if p.TimeoutProcessProposal < 1 {
		p.TimeoutProcessProposal = d.TimeoutProcessProposal
	}

	p.Threshold.Total = 1
	p.Threshold.Threshold = 1
	p.Threshold.Percent = cd.Threshold

	cd.PolicyOperationBodyV0 = p

	return nil
}

func (cd *ContestPolicyDesign) IsValid([]byte) error {
	if err := cd.PolicyOperationBodyV0.IsValid(nil); err != nil {
		return err
	}

	if cd.Threshold < 1 {
		return isvalid.InvalidError.Errorf("0 percent found: %v", cd.Threshold)
	} else if cd.Threshold > 100 {
		return isvalid.InvalidError.Errorf("over 100 percent: %v", cd.Threshold)
	}

	return nil
}
