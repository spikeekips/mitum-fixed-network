package element

import (
	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

type Block interface {
	Version() semver.Version
	Hash() common.Hash
	PrevHash() common.Hash
	State() []byte
	PrevState() []byte
	Transactions() []common.Hash
}
