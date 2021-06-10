package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/logging"
)

type baseBlocksValidationChecker struct {
	*logging.Logging
	networkID base.NetworkID
}

func (bc *baseBlocksValidationChecker) checkIsValid(blk block.Manifest) error {
	if blk == nil {
		return xerrors.Errorf("nil manifest found")
	}

	return blk.IsValid(bc.networkID)
}

func (*baseBlocksValidationChecker) checkPreviousBlock(current, next block.Manifest) error {
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
	networkID base.NetworkID,
	manifests []block.Manifest,
) *ManifestsValidationChecker {
	return &ManifestsValidationChecker{
		baseBlocksValidationChecker: baseBlocksValidationChecker{
			Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
				return c.
					Hinted("from_manifest", manifests[0].Height()).
					Hinted("to_manifest", manifests[len(manifests)-1].Height()).
					Str("module", "manifests-validation-checker")
			}),
			networkID: networkID,
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
			return false, NewBlockIntegrityError(current, err)
		}

		i++
	}

	bc.Log().Debug().Msg("validated serialized manifests")

	return true, nil
}
