package blockdata

import (
	"io"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

type Writer interface {
	hint.Hinter
	WriteManifest(io.Writer, block.Manifest) error
	WriteOperations(io.Writer, []operation.Operation) error
	WriteOperationsTree(io.Writer, tree.FixedTree) error
	WriteStates(io.Writer, []state.State) error
	WriteStatesTree(io.Writer, tree.FixedTree) error
	WriteINITVoteproof(io.Writer, base.Voteproof) error
	WriteACCEPTVoteproof(io.Writer, base.Voteproof) error
	WriteSuffrageInfo(io.Writer, block.SuffrageInfo) error
	WriteProposal(io.Writer, base.SignedBallotFact) error
	ReadManifest(io.Reader) (block.Manifest, error)
	ReadOperations(io.Reader) ([]operation.Operation, error)
	ReadOperationsTree(io.Reader) (tree.FixedTree, error)
	ReadStates(io.Reader) ([]state.State, error)
	ReadStatesTree(io.Reader) (tree.FixedTree, error)
	ReadINITVoteproof(io.Reader) (base.Voteproof, error)
	ReadACCEPTVoteproof(io.Reader) (base.Voteproof, error)
	ReadSuffrageInfo(io.Reader) (block.SuffrageInfo, error)
	ReadProposal(io.Reader) (base.SignedBallotFact, error)
}
