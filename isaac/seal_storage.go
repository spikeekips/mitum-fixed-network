package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type SealStorage interface {
	Has(hash.Hash /* seal.Seal.Hash() */) bool
	Get(hash.Hash /* seal.Seal.Hash() */) (seal.Seal, bool)
	GetProposal(node.Address /* proposer */, Height, Round) (Proposal, bool)
	Save(seal.Seal) error
}
