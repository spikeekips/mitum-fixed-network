package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/seal"
)

type SealStorage interface {
	Has(hash.Hash /* seal.Seal.Hash() */) bool
	Get(hash.Hash /* seal.Seal.Hash() */) seal.Seal
	Save(seal.Seal) error
}
