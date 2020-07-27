package block

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BlockV0Type              = hint.MustNewType(0x01, 0x40, "block-v0")
	BlockV0Hint              = hint.MustHint(BlockV0Type, "0.0.1")
	ManifestV0Type           = hint.MustNewType(0x01, 0x41, "block-manifest-v0")
	ManifestV0Hint           = hint.MustHint(ManifestV0Type, "0.0.1")
	BlockConsensusInfoV0Type = hint.MustNewType(0x01, 0x42, "block-consensus-info-v0")
	BlockConsensusInfoV0Hint = hint.MustHint(BlockConsensusInfoV0Type, "0.0.1")
	SuffrageInfoV0Type       = hint.MustNewType(0x01, 0x43, "block-suffrage-info-v0")
	SuffrageInfoV0Hint       = hint.MustHint(SuffrageInfoV0Type, "0.0.1")
)

type BlockV0 struct {
	ManifestV0
	BlockConsensusInfoV0
	operations *tree.AVLTree
	states     *tree.AVLTree
}

func NewBlockV0(
	si SuffrageInfoV0,
	height base.Height,
	round base.Round,
	proposal valuehash.Hash,
	previousBlock valuehash.Hash,
	operationsHash valuehash.Hash,
	statesHash valuehash.Hash,
) (BlockV0, error) {
	bm := ManifestV0{
		previousBlock:  previousBlock,
		height:         height,
		round:          round,
		proposal:       proposal,
		operationsHash: operationsHash,
		statesHash:     statesHash,
		createdAt:      localtime.Now(),
	}
	bm.h = bm.GenerateHash()

	return BlockV0{
		ManifestV0: bm,
		BlockConsensusInfoV0: BlockConsensusInfoV0{
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
			bm.BlockConsensusInfoV0,
		}, networkID, false); err != nil {
			return err
		}
	}

	if bm.OperationsHash() != nil {
		if bm.operations == nil {
			return xerrors.Errorf("Operations should not be empty")
		}

		if !bm.OperationsHash().Equal(bm.operations.RootHash()) {
			return xerrors.Errorf("Block.Opertions() hash does not match with it's RootHash()")
		}
	}

	if bm.StatesHash() != nil {
		if bm.states == nil {
			return xerrors.Errorf("States should not be empty")
		}

		if !bm.StatesHash().Equal(bm.States().RootHash()) {
			return xerrors.Errorf("Block.States() hash does not match with it's RootHash()")
		}
	}

	return nil
}

func (bm BlockV0) Hint() hint.Hint {
	return BlockV0Hint
}

func (bm BlockV0) Bytes() []byte {
	return nil
}

func (bm BlockV0) SetINITVoteproof(voteproof base.Voteproof) BlockUpdater {
	bm.BlockConsensusInfoV0.initVoteproof = voteproof

	return bm
}

func (bm BlockV0) SetACCEPTVoteproof(voteproof base.Voteproof) BlockUpdater {
	bm.BlockConsensusInfoV0.acceptVoteproof = voteproof

	return bm
}

func (bm BlockV0) Manifest() Manifest {
	return bm.ManifestV0
}

func (bm BlockV0) ConsensusInfo() BlockConsensusInfo {
	return bm.BlockConsensusInfoV0
}

func (bm BlockV0) Operations() *tree.AVLTree {
	return bm.operations
}

func (bm BlockV0) SetOperations(tr *tree.AVLTree) BlockUpdater {
	bm.operations = tr

	return bm
}

func (bm BlockV0) States() *tree.AVLTree {
	return bm.states
}

func (bm BlockV0) SetStates(tr *tree.AVLTree) BlockUpdater {
	bm.states = tr

	return bm
}
