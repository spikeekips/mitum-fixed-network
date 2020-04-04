package isaac

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"golang.org/x/xerrors"
)

type baseBlocksValidationChecker struct {
	*logging.Logging
	localstate *Localstate
	networkID  NetworkID
}

func (bc *baseBlocksValidationChecker) checkIsValid(block Manifest) error {
	return block.IsValid(bc.networkID)
}

func (bc *baseBlocksValidationChecker) checkPreviousBlock(current, next Manifest) error {
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
	manifests []Manifest
}

func NewManifestsValidationChecker(
	localstate *Localstate,
	manifests []Manifest,
) *ManifestsValidationChecker {
	networkID := localstate.Policy().NetworkID()

	return &ManifestsValidationChecker{
		baseBlocksValidationChecker: baseBlocksValidationChecker{
			Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
				return c.
					Int64("from_manifest", manifests[0].Height().Int64()).
					Int64("to_manifest", manifests[len(manifests)-1].Height().Int64()).
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
	blocks []Block
}

func NewBlocksValidationChecker(localstate *Localstate, blocks []Block) *BlocksValidationChecker {
	networkID := localstate.Policy().NetworkID()

	return &BlocksValidationChecker{
		baseBlocksValidationChecker: baseBlocksValidationChecker{
			Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
				return c.
					Int64("from_block", blocks[0].Height().Int64()).
					Int64("to_block", blocks[len(blocks)-1].Height().Int64()).
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
