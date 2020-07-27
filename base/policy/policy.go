package policy

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

const PolicyOperationKey = "_network_policy"

var (
	DefaultPolicyThresholdRatio                   = base.ThresholdRatio(100)
	DefaultPolicyNumberOfActingSuffrageNodes      = uint(1)
	DefaultPolicyMaxOperationsInSeal         uint = 100
	DefaultPolicyMaxOperationsInProposal     uint = 100
)

type Policy interface {
	hint.Hinter
	valuehash.Hasher
	util.Byter
	isvalid.IsValider
	ThresholdRatio() base.ThresholdRatio
	NumberOfActingSuffrageNodes() uint
	MaxOperationsInSeal() uint
	MaxOperationsInProposal() uint
}
