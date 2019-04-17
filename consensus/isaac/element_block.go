package isaac

import (
	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

var (
	CurrentBlockVersion semver.Version = *semver.MustParse("v0.1-proto")
)

type Block struct {
	version semver.Version

	hash     common.Hash
	prevHash common.Hash

	state     []byte
	prevState []byte

	proposer common.Address
	proposed common.Time

	ballot       common.Hash
	transactions []common.Hash
}

func (b Block) Version() semver.Version {
	return b.version
}

func (b Block) Hash() common.Hash {
	return b.hash
}

func (b Block) PrevHash() common.Hash {
	return b.prevHash
}

func (b Block) State() []byte {
	return b.state
}

func (b Block) PrevState() []byte {
	return b.prevState
}

func (b Block) Transactions() []common.Hash {
	return b.transactions
}
