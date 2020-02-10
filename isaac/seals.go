package isaac

import (
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type SealStorage interface {
	Add(seal.Seal) error
	Delete(valuehash.Hash) error
	Exists(valuehash.Hash) (bool, error)
	Seal(valuehash.Hash) (seal.Seal, bool, error)
	Proposal(Height, Round) (Proposal, bool, error)
}
