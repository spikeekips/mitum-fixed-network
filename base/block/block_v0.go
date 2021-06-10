package block

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BlockV0Type              = hint.Type("block")
	BlockV0Hint              = hint.NewHint(BlockV0Type, "v0.0.1")
	ManifestV0Type           = hint.Type("block-manifest")
	ManifestV0Hint           = hint.NewHint(ManifestV0Type, "v0.0.1")
	BlockConsensusInfoV0Type = hint.Type("block-consensus-info")
	BlockConsensusInfoV0Hint = hint.NewHint(BlockConsensusInfoV0Type, "v0.0.1")
	SuffrageInfoV0Type       = hint.Type("block-suffrage-info")
	SuffrageInfoV0Hint       = hint.NewHint(SuffrageInfoV0Type, "v0.0.1")
)

type BlockV0 struct {
	ManifestV0
	operationsTree tree.FixedTree
	operations     []operation.Operation
	statesTree     tree.FixedTree
	states         []state.State
	ci             ConsensusInfoV0
}

func NewBlockV0(
	si SuffrageInfoV0,
	height base.Height,
	round base.Round,
	proposal valuehash.Hash,
	previousBlock valuehash.Hash,
	operationsHash valuehash.Hash,
	statesHash valuehash.Hash,
	confirmedAt time.Time,
) (BlockV0, error) {
	bm := ManifestV0{
		previousBlock:  previousBlock,
		height:         height,
		round:          round,
		proposal:       proposal,
		operationsHash: operationsHash,
		statesHash:     statesHash,
		confirmedAt:    confirmedAt,
		createdAt:      localtime.UTCNow(),
	}
	bm.h = bm.GenerateHash()

	return BlockV0{
		ManifestV0: bm,
		ci: ConsensusInfoV0{
			suffrageInfo: si,
		},
	}, nil
}

func (bm BlockV0) IsValid(networkID []byte) error {
	if bm.height == base.PreGenesisHeight {
		if err := isvalid.Check([]isvalid.IsValider{bm.ManifestV0}, networkID, false); err != nil {
			return err
		}
	} else {
		if err := isvalid.Check([]isvalid.IsValider{
			bm.ManifestV0,
			bm.ci,
		}, networkID, false); err != nil {
			return err
		}
	}

	if bm.operationsHash == nil || bm.operationsHash.Empty() {
		bm.operationsHash = nil
	}
	if bm.statesHash == nil || bm.statesHash.Empty() {
		bm.statesHash = nil
	}

	if bm.operationsHash != nil {
		if bm.operations == nil || bm.operationsTree.Len() < 1 {
			return xerrors.Errorf("Operations should not be empty")
		}

		if !bm.operationsHash.Equal(valuehash.NewBytes(bm.operationsTree.Root())) {
			return xerrors.Errorf("Block.Opertions() hash does not match with it's Root()")
		}
	}

	if bm.statesHash != nil {
		if bm.states == nil || bm.statesTree.Len() < 1 {
			return xerrors.Errorf("States should not be empty")
		}

		if !bm.statesHash.Equal(valuehash.NewBytes(bm.statesTree.Root())) {
			return xerrors.Errorf("Block.States() hash does not match with it's Root()")
		}
	}

	return nil
}

func (BlockV0) Hint() hint.Hint {
	return BlockV0Hint
}

func (BlockV0) Bytes() []byte {
	return nil
}

func (bm BlockV0) SetINITVoteproof(voteproof base.Voteproof) BlockUpdater {
	bm.ci.initVoteproof = voteproof

	return bm
}

func (bm BlockV0) SetACCEPTVoteproof(voteproof base.Voteproof) BlockUpdater {
	bm.ci.acceptVoteproof = voteproof

	return bm
}

func (bm BlockV0) SetSuffrageInfo(sf SuffrageInfo) BlockUpdater {
	bm.ci.suffrageInfo = sf

	return bm
}

func (bm BlockV0) SetProposal(proposal ballot.Proposal) BlockUpdater {
	bm.ci.proposal = proposal

	return bm
}

func (bm BlockV0) SetManifest(m Manifest) BlockUpdater {
	bm.ManifestV0 = m.(ManifestV0)

	return bm
}

func (bm BlockV0) Manifest() Manifest {
	return bm.ManifestV0
}

func (bm BlockV0) ConsensusInfo() ConsensusInfo {
	return bm.ci
}

func (bm BlockV0) OperationsTree() tree.FixedTree {
	return bm.operationsTree
}

func (bm BlockV0) Operations() []operation.Operation {
	return bm.operations
}

func (bm BlockV0) SetOperationsTree(tr tree.FixedTree) BlockUpdater {
	bm.operationsTree = tr

	return bm
}

func (bm BlockV0) SetOperations(ops []operation.Operation) BlockUpdater {
	bm.operations = ops

	return bm
}

func (bm BlockV0) StatesTree() tree.FixedTree {
	return bm.statesTree
}

func (bm BlockV0) SetStatesTree(tr tree.FixedTree) BlockUpdater {
	bm.statesTree = tr

	return bm
}

func (bm BlockV0) States() []state.State {
	return bm.states
}

func (bm BlockV0) SetStates(sts []state.State) BlockUpdater {
	bm.states = sts

	return bm
}
