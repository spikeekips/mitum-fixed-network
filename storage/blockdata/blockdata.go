package blockdata

import (
	"io"
	"io/fs"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

type BlockData interface {
	hint.Hinter
	Initialize() error
	IsLocal() bool
	Exists(base.Height) (bool, error)
	Remove(base.Height) error
	RemoveAll(base.Height) error
	Clean(remove bool) error
	NewSession(base.Height) (Session, error)
	SaveSession(Session) (block.BlockDataMap, error)
	FS() fs.FS
	Writer() Writer
}

type Session interface {
	Height() base.Height
	SetBlock(block.Block) error
	SetManifest(block.Manifest) error
	AddOperations(...operation.Operation) error
	CloseOperations() error
	SetOperationsTree(tree.FixedTree) error
	AddStates(...state.State) error
	CloseStates() error
	SetStatesTree(tree.FixedTree) error
	SetINITVoteproof(base.Voteproof) error
	SetACCEPTVoteproof(base.Voteproof) error
	SetSuffrageInfo(block.SuffrageInfo) error
	SetProposal(base.SignedBallotFact) error
	Import(string, io.Reader) (string /* file path */, error)
	Cancel() error
}
