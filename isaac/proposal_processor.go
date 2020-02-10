package isaac

import "github.com/spikeekips/mitum/valuehash"

type ProposalProcessor interface {
	Process(valuehash.Hash /* Proposal.Hash() */, []byte) (Block, error)
}
