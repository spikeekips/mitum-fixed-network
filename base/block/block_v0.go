package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BlockV0Type                = hint.Type("block")
	BlockV0Hint                = hint.NewHint(BlockV0Type, "v0.0.1")
	BlockV0Hinter              = BlockV0{BaseHinter: hint.NewBaseHinter(BlockV0Hint)}
	ManifestV0Type             = hint.Type("block-manifest")
	ManifestV0Hint             = hint.NewHint(ManifestV0Type, "v0.0.1")
	ManifestV0Hinter           = ManifestV0{BaseHinter: hint.NewBaseHinter(ManifestV0Hint)}
	BlockConsensusInfoV0Type   = hint.Type("block-consensus-info")
	BlockConsensusInfoV0Hint   = hint.NewHint(BlockConsensusInfoV0Type, "v0.0.1")
	BlockConsensusInfoV0Hinter = ConsensusInfoV0{BaseHinter: hint.NewBaseHinter(BlockConsensusInfoV0Hint)}
	SuffrageInfoV0Type         = hint.Type("block-suffrage-info")
	SuffrageInfoV0Hint         = hint.NewHint(SuffrageInfoV0Type, "v0.0.1")
	SuffrageInfoV0Hinter       = SuffrageInfoV0{BaseHinter: hint.NewBaseHinter(SuffrageInfoV0Hint)}
)

type BlockV0 struct {
	hint.BaseHinter
	ManifestV0
	operationsTree tree.FixedTree
	operations     []operation.Operation
	statesTree     tree.FixedTree
	states         []state.State
	ci             ConsensusInfoV0
}

func EmptyBlockV0() BlockV0 {
	return BlockV0{
		BaseHinter: hint.NewBaseHinter(BlockV0Hint),
		ci: ConsensusInfoV0{
			BaseHinter: hint.NewBaseHinter(BlockConsensusInfoV0Hint),
		},
	}
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
		BaseHinter:     hint.NewBaseHinter(ManifestV0Hint),
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
		BaseHinter: hint.NewBaseHinter(BlockV0Hint),
		ManifestV0: bm,
		ci: ConsensusInfoV0{
			BaseHinter:   hint.NewBaseHinter(BlockConsensusInfoV0Hint),
			suffrageInfo: si,
		},
	}, nil
}

func (bm BlockV0) IsValid(networkID []byte) error {
	if err := bm.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if bm.height == base.PreGenesisHeight {
		if err := isvalid.Check(networkID, false, bm.ManifestV0); err != nil {
			return err
		}
	} else if err := isvalid.Check(networkID, false, bm.ManifestV0, bm.ci); err != nil {
		return err
	}

	if bm.operationsHash != nil && !bm.operationsHash.IsEmpty() {
		if bm.operations == nil || bm.operationsTree.Len() < 1 {
			return isvalid.InvalidError.Errorf("Operations should not be empty")
		}

		if !bm.operationsHash.Equal(valuehash.NewBytes(bm.operationsTree.Root())) {
			return isvalid.InvalidError.Errorf("Block.Operations() hash does not match with it's Root()")
		}
	}

	if bm.statesHash != nil && !bm.statesHash.IsEmpty() {
		if bm.states == nil || bm.statesTree.Len() < 1 {
			return isvalid.InvalidError.Errorf("States should not be empty")
		}

		if !bm.statesHash.Equal(valuehash.NewBytes(bm.statesTree.Root())) {
			return isvalid.InvalidError.Errorf("Block.States() hash does not match with it's Root()")
		}
	}

	if bm.proposal != nil && bm.ci.Proposal() != nil && !bm.proposal.Equal(bm.ci.Proposal().Fact().Hash()) {
		return isvalid.InvalidError.Errorf("proposal does not match with consensus info")
	}

	return nil
}

func (bm BlockV0) Hint() hint.Hint {
	return bm.BaseHinter.Hint()
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

func (bm BlockV0) SetProposal(sfs base.SignedBallotFact) BlockUpdater {
	bm.ci.sfs = sfs

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
