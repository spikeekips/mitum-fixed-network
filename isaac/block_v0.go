package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/tree"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

var (
	BlockV0Type              = hint.MustNewType(0x05, 0x00, "block-v0")
	BlockV0Hint              = hint.MustHint(BlockV0Type, "0.0.1")
	BlockManifestV0Type      = hint.MustNewType(0x05, 0x01, "block-manifest-v0")
	BlockManifestV0Hint      = hint.MustHint(BlockManifestV0Type, "0.0.1")
	BlockConsensusInfoV0Type = hint.MustNewType(0x05, 0x02, "block-consensus-info-v0")
	BlockConsensusInfoV0Hint = hint.MustHint(BlockConsensusInfoV0Type, "0.0.1")
)

type BlockV0 struct {
	BlockManifestV0
	BlockConsensusInfoV0
	operations *tree.AVLTree
	states     *tree.AVLTree
}

func NewBlockV0(
	height Height,
	round Round,
	proposal valuehash.Hash,
	previousBlock valuehash.Hash,
	operationsHash valuehash.Hash,
	statesHash valuehash.Hash,
) (BlockV0, error) {
	bm := BlockManifestV0{
		previousBlock:  previousBlock,
		height:         height,
		round:          round,
		proposal:       proposal,
		operationsHash: operationsHash,
		statesHash:     statesHash,
		createdAt:      localtime.Now(),
	}
	if h, err := bm.GenerateHash(); err != nil {
		return BlockV0{}, err
	} else {
		bm.h = h
	}

	return BlockV0{
		BlockManifestV0:      bm,
		BlockConsensusInfoV0: BlockConsensusInfoV0{},
	}, nil
}

func (bm BlockV0) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		bm.BlockManifestV0,
		bm.BlockConsensusInfoV0,
	}, nil, false); err != nil {
		return err
	}

	if bm.OperationsHash() != nil {
		if bm.operations == nil {
			return xerrors.Errorf("Operations should not be empty")
		}

		if rh, err := bm.operations.RootHash(); err != nil {
			return err
		} else if !bm.OperationsHash().Equal(rh) {
			return xerrors.Errorf("Block.Opertions() hash does not match with it's RootHash()")
		}
	}

	if bm.StatesHash() != nil {
		if bm.states == nil {
			return xerrors.Errorf("States should not be empty")
		}

		if rh, err := bm.States().RootHash(); err != nil {
			return err
		} else if !bm.StatesHash().Equal(rh) {
			return xerrors.Errorf("Block.States() hash does not match with it's RootHash()")
		}
	}

	return nil
}

func (bm BlockV0) Hint() hint.Hint {
	return BlockV0Hint
}

func (bm BlockV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		bm.BlockManifestV0.Bytes(),
		bm.BlockConsensusInfoV0.Bytes(),
	})
}

func (bm BlockV0) SetINITVoteproof(voteproof Voteproof) BlockUpdater {
	bm.BlockConsensusInfoV0.initVoteproof = voteproof

	return bm
}

func (bm BlockV0) SetACCEPTVoteproof(voteproof Voteproof) BlockUpdater {
	bm.BlockConsensusInfoV0.acceptVoteproof = voteproof

	return bm
}

func (bm BlockV0) Manifest() BlockManifest {
	return bm.BlockManifestV0
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
