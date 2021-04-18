package cmds

import (
	"context"
	"math"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"
)

type BaseVerifyCommand struct {
	*BaseCommand
	NetworkID  NetworkIDFlag `name:"network-id"`
	networkID  base.NetworkID
	lastHeight base.Height
}

func NewBaseVerifyCommand(name string, hinters []hint.Hinter) *BaseVerifyCommand {
	b := NewBaseCommand(name)
	if _, err := b.LoadEncoders(hinters); err != nil {
		panic(err)
	}

	return &BaseVerifyCommand{
		BaseCommand: b,
	}
}

func (cmd *BaseVerifyCommand) Initialize(flags interface{}, version util.Version) error {
	if err := cmd.BaseCommand.Initialize(flags, version); err != nil {
		return err
	}

	if len(cmd.NetworkID) < 1 {
		cmd.Log().Warn().Msg("empty network-id")
	} else {
		cmd.networkID = cmd.NetworkID.NetworkID()
	}

	return nil
}

func (cmd *BaseVerifyCommand) checkAllManifests(
	get func(base.Height) (block.Manifest, error),
) error {
	cmd.Log().Info().Msg("checking manifests")

	lh := cmd.lastHeight.Int64() + 1
	limit := int64(50)
	c := int64(math.Ceil(float64(lh) / float64(limit)))

	var baseManifest block.Manifest
	for i := int64(0); i < c; i++ {
		s := i * limit
		e := (i + 1) * limit
		if i == 0 {
			s = -1
			e = limit
		}

		if e > lh {
			e = lh
		}

		if j, err := cmd.checkManifests(baseManifest, base.Height(s), base.Height(e), get); err != nil {
			return err
		} else {
			baseManifest = j
		}
	}

	return nil
}

func (cmd *BaseVerifyCommand) checkManifests(
	base block.Manifest,
	s, e base.Height,
	get func(base.Height) (block.Manifest, error),
) (block.Manifest, error) {
	l := cmd.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		e := ctx.Ints64("heights", []int64{s.Int64(), e.Int64()})

		if base != nil {
			e = e.Int64("base_height", base.Height().Int64())
		}

		return e
	})

	var manifests []block.Manifest
	if i, err := cmd.loadManifests(s, e, get); err != nil {
		return nil, xerrors.Errorf("failed to load manifests, %d-%d: %w", s, e, err)
	} else {
		if base == nil {
			manifests = i
		} else {
			manifests = make([]block.Manifest, len(i)+1)
			manifests[0] = base
			copy(manifests[1:], i)
		}

		l.Debug().Msg("manifests loaded")
	}

	checker := isaac.NewManifestsValidationChecker(cmd.networkID, manifests)
	if err := util.NewChecker("manifests-validation-checker", []util.CheckerFunc{
		checker.CheckSerialized,
	}).Check(); err != nil {
		l.Error().Err(err).Msg("failed to verify manifests")

		return nil, err
	}

	l.Debug().Msg("manifests checked")
	return manifests[len(manifests)-1], nil
}

func (cmd *BaseVerifyCommand) loadManifests(
	s, e base.Height,
	get func(base.Height) (block.Manifest, error),
) ([]block.Manifest, error) {
	mch := make(chan block.Manifest)
	errch := make(chan error)

	eg, ctx := errgroup.WithContext(context.Background())
	go func() {
		sem := semaphore.NewWeighted(100)

		for i := s; i < e; i++ {
			height := i
			if err := sem.Acquire(ctx, 1); err != nil {
				break
			}

			eg.Go(func() error {
				defer sem.Release(1)

				if j, err := get(height); err != nil {
					cmd.Log().Error().Err(err).Int64("height", height.Int64()).Msg("failed to load manifest")
				} else {
					mch <- j
				}

				return nil
			})
		}

		if err := sem.Acquire(ctx, 100); err != nil {
			errch <- err
		} else if err := eg.Wait(); err != nil {
			errch <- err
		}
	}()

	manifests := make([]block.Manifest, (e - s).Int64())

end:
	for {
		select {
		case <-ctx.Done():
			break end
		case err := <-errch:
			return nil, err
		case i := <-mch:
			manifests[(i.Height() - s).Int64()] = i
		}
	}

	return manifests, nil
}
