package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type ProposalProcessor interface {
	util.Initializer
	IsProcessed(valuehash.Hash /* proposal.Hash() */) bool
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, base.Voteproof /* INIT Voteproof */) (block.Block, error)
	ProcessACCEPT(
		valuehash.Hash /* Proposal.Hash() */, base.Voteproof, /* ACCEPT Voteproof */
	) (storage.BlockStorage, error)
}
