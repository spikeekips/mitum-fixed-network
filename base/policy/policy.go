package policy

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

const PolicyOperationKey = "_network_policy"

var (
	DefaultPolicyNumberOfActingSuffrageNodes      = uint(1)
	DefaultPolicyMaxOperationsInSeal         uint = 100
	DefaultPolicyMaxOperationsInProposal     uint = 100
)

type Policy interface {
	hint.Hinter
	valuehash.Hasher
	util.Byter
	isvalid.IsValider
	NumberOfActingSuffrageNodes() uint // TODO NumberOfActingSuffrageNodes should be moved under roundrobin config
	MaxOperationsInSeal() uint
	MaxOperationsInProposal() uint
}
