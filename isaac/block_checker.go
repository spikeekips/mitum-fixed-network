package isaac

import (
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type baseBlocksValidationChecker struct {
	*logging.Logging
	localstate *Localstate
	networkID  NetworkID
}

func (bc *baseBlocksValidationChecker) checkIsValid(blk block.Manifest) error {
	return blk.IsValid(bc.networkID)
}

func (bc *baseBlocksValidationChecker) checkPreviousBlock(current, next block.Manifest) error {
	if next.Height() != current.Height()+1 {
		return xerrors.Errorf("wrong height: current=%v next=%s", current.Height(), next.Height())
	}

	if !next.PreviousBlock().Equal(current.Hash()) {
		return xerrors.Errorf(
			"chained Hash does not match: height=%v current=%s next=%s",
			next.Height(),
			current.Hash(),
			next.PreviousBlock(),
		)
	}

	return nil
}

type ManifestsValidationChecker struct {
	baseBlocksValidationChecker
	manifests []block.Manifest
}

func NewManifestsValidationChecker(
	localstate *Localstate,
	manifests []block.Manifest,
) *ManifestsValidationChecker {
	networkID := localstate.Policy().NetworkID()

	return &ManifestsValidationChecker{
		baseBlocksValidationChecker: baseBlocksValidationChecker{
			Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
				return c.
					Hinted("from_manifest", manifests[0].Height()).
					Hinted("to_manifest", manifests[len(manifests)-1].Height()).
					Str("module", "manifests-validation-checker")
			}),
			localstate: localstate,
			networkID:  networkID,
		},
		manifests: manifests,
	}
}

func (bc *ManifestsValidationChecker) CheckSerialized() (bool, error) {
	bc.Log().Debug().Msg("trying to validate serialized manifests")

	i := 0
	l := len(bc.manifests)
	for {
		current := bc.manifests[i]
		if err := bc.checkIsValid(current); err != nil {
			return false, err
		}

		if l == i+1 {
			break
		}

		if err := bc.checkPreviousBlock(current, bc.manifests[i+1]); err != nil {
			return false, err
		}

		i++
	}

	bc.Log().Debug().Msg("validated serialized manifests")

	return true, nil
}

type BlocksValidationChecker struct {
	baseBlocksValidationChecker
	blocks []block.Block
}

func NewBlocksValidationChecker(localstate *Localstate, blocks []block.Block) *BlocksValidationChecker {
	networkID := localstate.Policy().NetworkID()

	return &BlocksValidationChecker{
		baseBlocksValidationChecker: baseBlocksValidationChecker{
			Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
				return c.
					Hinted("from_block", blocks[0].Height()).
					Hinted("to_block", blocks[len(blocks)-1].Height()).
					Str("module", "blocks-validation-checker")
			}),
			localstate: localstate,
			networkID:  networkID,
		},
		blocks: blocks,
	}
}

func (bc *BlocksValidationChecker) CheckSerialized() (bool, error) {
	bc.Log().Debug().Msg("trying to validate serialized blocks")

	i := 0
	l := len(bc.blocks)
	for {
		current := bc.blocks[i]
		if err := bc.checkIsValid(current); err != nil {
			return false, err
		}

		if l == i+1 {
			break
		}

		if err := bc.checkPreviousBlock(current, bc.blocks[i+1]); err != nil {
			return false, err
		}

		i++
	}

	bc.Log().Debug().Msg("validated serialized blocks")

	return true, nil
}

// TODO Block has suffrage infor and consensus info, so it's voteproof should be
// validated by suffrage info.
