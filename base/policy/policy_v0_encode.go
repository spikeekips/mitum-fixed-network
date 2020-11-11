package policy

func (po *PolicyV0) unpack(
	numberOfActingSuffrageNodes uint,
	maxOperationsInSeal uint,
	maxOperationsInProposal uint,
) error {
	po.numberOfActingSuffrageNodes = numberOfActingSuffrageNodes
	po.maxOperationsInSeal = maxOperationsInSeal
	po.maxOperationsInProposal = maxOperationsInProposal

	return nil
}
