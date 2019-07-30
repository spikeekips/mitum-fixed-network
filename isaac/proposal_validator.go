package isaac

import "github.com/spikeekips/mitum/hash"

type ProposalValidator interface {
	Validated(hash.Hash /* Proposal.Hash() */) bool
	NewBlock(Height, Round, hash.Hash /* Proposal.Hash() */) (Block, error)
}
