package cmds

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type BaseVerifyCommand struct {
	*BaseCommand
	NetworkID  NetworkIDFlag `name:"network-id"`
	networkID  base.NetworkID
	lastHeight base.Height
}

func NewBaseVerifyCommand(name string, types []hint.Type, hinters []hint.Hinter) *BaseVerifyCommand {
	b := NewBaseCommand(name)
	if _, err := b.LoadEncoders(types, hinters); err != nil {
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

		j, err := cmd.checkManifests(baseManifest, base.Height(s), base.Height(e), get)
		if err != nil {
			return err
		}
		baseManifest = j
	}

	return nil
}

func (cmd *BaseVerifyCommand) checkManifests(
	b block.Manifest,
	s, e base.Height,
	get func(base.Height) (block.Manifest, error),
) (block.Manifest, error) {
	var l zerolog.Logger
	{
		c := cmd.Log().With().Ints64("heights", []int64{s.Int64(), e.Int64()})
		if b != nil {
			c = c.Int64("base_height", b.Height().Int64())
		}
		l = c.Logger()
	}

	var manifests []block.Manifest
	i, err := cmd.loadManifests(s, e, get)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load manifests, %d-%d", s, e)
	}

	if b == nil {
		manifests = i
	} else {
		manifests = make([]block.Manifest, len(i)+1)
		manifests[0] = b
		copy(manifests[1:], i)
	}

	l.Debug().Msg("manifests loaded")

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
			if !errors.Is(err, context.Canceled) {
				errch <- err
			}
		}

		if err := eg.Wait(); err != nil {
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
