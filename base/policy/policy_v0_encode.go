package policy

import (
	"github.com/spikeekips/mitum/base"
)

func (po *PolicyV0) unpack(
	thresholdRatio base.ThresholdRatio,
	numberOfActingSuffrageNodes uint,
	maxOperationsInSeal uint,
	maxOperationsInProposal uint,
) error {
	po.thresholdRatio = thresholdRatio
	po.numberOfActingSuffrageNodes = numberOfActingSuffrageNodes
	po.maxOperationsInSeal = maxOperationsInSeal
	po.maxOperationsInProposal = maxOperationsInProposal

	return nil
}
