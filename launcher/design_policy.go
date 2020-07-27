package launcher

import (
	"github.com/spikeekips/mitum/base/policy"
)

type PolicyDesign struct {
	policy.PolicyV0
}

func NewPolicyDesign() *PolicyDesign {
	return &PolicyDesign{
		PolicyV0: policy.DefaultPolicyV0(),
	}
}

func (cd PolicyDesign) Policy() policy.Policy {
	return cd.PolicyV0
}

func (cd *PolicyDesign) IsValid([]byte) error {
	po := cd.PolicyV0
	if po.ThresholdRatio() < 1 {
		po = po.SetThresholdRatio(policy.DefaultPolicyThresholdRatio)
	}
	if po.NumberOfActingSuffrageNodes() < 1 {
		po = po.SetNumberOfActingSuffrageNodes(policy.DefaultPolicyNumberOfActingSuffrageNodes)
	}
	if po.MaxOperationsInSeal() < 1 {
		po = po.SetMaxOperationsInSeal(policy.DefaultPolicyMaxOperationsInSeal)
	}
	if po.MaxOperationsInProposal() < 1 {
		po = po.SetMaxOperationsInProposal(policy.DefaultPolicyMaxOperationsInProposal)
	}

	cd.PolicyV0 = po

	return cd.PolicyV0.IsValid(nil)
}
