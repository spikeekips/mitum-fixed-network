package policy

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	PolicyV0Type = hint.MustNewType(0x01, 0x0b, "policy-v0")
	PolicyV0Hint = hint.MustHint(PolicyV0Type, "0.0.1")
)

type PolicyV0 struct {
	numberOfActingSuffrageNodes uint
	maxOperationsInSeal         uint
	maxOperationsInProposal     uint
}

func DefaultPolicyV0() PolicyV0 {
	return NewPolicyV0(
		DefaultPolicyNumberOfActingSuffrageNodes,
		DefaultPolicyMaxOperationsInSeal,
		DefaultPolicyMaxOperationsInProposal,
	)
}

func NewPolicyV0(
	numberOfActingSuffrageNodes uint,
	maxOperationsInSeal uint,
	maxOperationsInProposal uint,
) PolicyV0 {
	return PolicyV0{
		numberOfActingSuffrageNodes: numberOfActingSuffrageNodes,
		maxOperationsInSeal:         maxOperationsInSeal,
		maxOperationsInProposal:     maxOperationsInProposal,
	}
}

func (po PolicyV0) Hint() hint.Hint {
	return PolicyV0Hint
}

func (po PolicyV0) IsValid([]byte) error {
	if po.numberOfActingSuffrageNodes < 1 {
		return xerrors.Errorf("NumberOfActingSuffrageNodes must be over 0; %d", po.numberOfActingSuffrageNodes)
	}
	if po.maxOperationsInSeal < 1 {
		return xerrors.Errorf("MaxOperationsInSeal must be over 0; %d", po.maxOperationsInSeal)
	}
	if po.maxOperationsInProposal < 1 {
		return xerrors.Errorf("MaxOperationsInProposal must be over 0; %d", po.MaxOperationsInProposal)
	}

	return nil
}

func (po PolicyV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		util.UintToBytes(po.numberOfActingSuffrageNodes),
		util.UintToBytes(po.maxOperationsInSeal),
		util.UintToBytes(po.maxOperationsInProposal),
	)
}

func (po PolicyV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(po.Bytes())
}

func (po PolicyV0) NumberOfActingSuffrageNodes() uint {
	return po.numberOfActingSuffrageNodes
}

func (po PolicyV0) SetNumberOfActingSuffrageNodes(n uint) PolicyV0 {
	po.numberOfActingSuffrageNodes = n

	return po
}

func (po PolicyV0) MaxOperationsInSeal() uint {
	return po.maxOperationsInSeal
}

func (po PolicyV0) SetMaxOperationsInSeal(m uint) PolicyV0 {
	po.maxOperationsInSeal = m

	return po
}

func (po PolicyV0) MaxOperationsInProposal() uint {
	return po.maxOperationsInProposal
}

func (po PolicyV0) SetMaxOperationsInProposal(m uint) PolicyV0 {
	po.maxOperationsInProposal = m

	return po
}
