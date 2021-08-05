package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"
)

type BlockDataVerifyCommand struct {
	*BaseVerifyCommand
	Path string `arg:"" name:"blockdata path"`
	bd   blockdata.BlockData
}

func NewBlockDataVerifyCommand(types []hint.Type, hinters []hint.Hinter) BlockDataVerifyCommand {
	return BlockDataVerifyCommand{
		BaseVerifyCommand: NewBaseVerifyCommand("blockdata-verify", types, hinters),
	}
}

func (cmd *BlockDataVerifyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	cmd.Log().Debug().Str("path", cmd.Path).Msg("trying to verify blockdata")

	return cmd.verify()
}

func (cmd *BlockDataVerifyCommand) Initialize(flags interface{}, version util.Version) error {
	if err := cmd.BaseVerifyCommand.Initialize(flags, version); err != nil {
		return err
	}

	if i, err := os.Stat(cmd.Path); err != nil {
		return xerrors.Errorf("invalid path, %q: %w", cmd.Path, err)
	} else if !i.IsDir() {
		return xerrors.Errorf("path, %q is not directory", cmd.Path)
	}

	cmd.bd = localfs.NewBlockData(cmd.Path, cmd.jsonenc)
	return cmd.bd.Initialize()
}

func (cmd *BlockDataVerifyCommand) verify() error {
	if err := cmd.checkLastHeight(); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to check last height")

		return err
	} else if cmd.lastHeight < base.PreGenesisHeight {
		return nil
	}

	var hasError bool
	if err := cmd.checkAllManifests(cmd.loadManifest); err != nil {
		hasError = true
	}

	if err := cmd.checkAllBlockFiles(); err != nil {
		hasError = true
	}

	if err := cmd.checkBlocks(); err != nil {
		hasError = true
	}

	if hasError {
		cmd.Log().Error().Msg("failed to verify blockdata")
	} else {
		cmd.Log().Debug().Msg("blockdata verified")
	}

	return nil
}

func (cmd *BlockDataVerifyCommand) checkLastHeight() error {
	var height base.Height = base.PreGenesisHeight
	for {
		if found, err := cmd.bd.Exists(height); err != nil {
			return xerrors.Errorf("failed to check blockdata of height, %d: %w", height, err)
		} else if !found {
			break
		}

		height++
	}

	cmd.lastHeight = height - 1

	cmd.Log().Info().Int64("last_height", cmd.lastHeight.Int64()).Msg("last height found")
	if cmd.lastHeight < base.PreGenesisHeight {
		cmd.Log().Warn().Msg("empty blockdata found")
	}

	return nil
}

func (cmd *BlockDataVerifyCommand) loadManifest(height base.Height) (block.Manifest, error) {
	var manifest block.Manifest
	_, i, err := localfs.LoadData(cmd.bd.(*localfs.BlockData), height, block.BlockDataManifest)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = i.Close()
	}()

	if j, err := cmd.bd.Writer().ReadManifest(i); err != nil {
		return nil, err
	} else if err := j.IsValid(cmd.networkID); err != nil {
		return nil, xerrors.Errorf("invalid manifest, %q found: %w", height, err)
	} else {
		manifest = j
	}

	return manifest, nil
}

func (cmd *BlockDataVerifyCommand) checkBlocks() error {
	sem := semaphore.NewWeighted(100)
	eg, ctx := errgroup.WithContext(context.Background())

	for i := base.PreGenesisHeight; i <= cmd.lastHeight; i++ {
		height := i
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		eg.Go(func() error {
			defer sem.Release(1)

			_, err := cmd.loadBlock(height)
			return err
		})
	}

	if err := sem.Acquire(ctx, 100); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	return eg.Wait()
}

func (cmd *BlockDataVerifyCommand) loadBlock(height base.Height) (block.Block, error) {
	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	if _, i, err := localfs.LoadBlock(cmd.bd.(*localfs.BlockData), height); err != nil {
		l.Error().Err(err).Msg("failed to load block")

		return nil, err
	} else if err := i.IsValid(cmd.networkID); err != nil {
		l.Error().Err(err).Msg("invalid block")

		return nil, err
	} else {
		l.Debug().Msg("block checked")

		return i, nil
	}
}

func (cmd *BlockDataVerifyCommand) checkAllBlockFiles() error {
	sem := semaphore.NewWeighted(100)
	eg, ctx := errgroup.WithContext(context.Background())

	for i := base.PreGenesisHeight; i <= cmd.lastHeight; i++ {
		height := i
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		eg.Go(func() error {
			defer sem.Release(1)

			return cmd.checkBlockFiles(height)
		})
	}

	if err := sem.Acquire(ctx, 100); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	return eg.Wait()
}

func (cmd *BlockDataVerifyCommand) checkBlockFiles(height base.Height) error {
	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	if found, err := cmd.bd.Exists(height); err != nil {
		return err
	} else if !found {
		return util.NotFoundError.Errorf("block data %d not found", height)
	}

	var hasError bool
	for i := range block.BlockData {
		dataType := block.BlockData[i]
		if err := cmd.checkBlockFile(height, dataType); err != nil {
			l.Error().Err(err).
				Int64("height", height.Int64()).
				Str("data_type", dataType).
				Msg("failed to check block data file")

			hasError = true
		}
	}

	if hasError {
		return xerrors.Errorf("block data file of height, %d has problem", height)
	}
	l.Debug().Msg("block data files checked")

	return nil
}

func (cmd *BlockDataVerifyCommand) checkBlockFile(height base.Height, dataType string) error {
	g := filepath.Join(cmd.Path, localfs.HeightDirectory(height), fmt.Sprintf("%d-%s-*.jsonld.gz", height, dataType))

	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return storage.WrapStorageError(err)
	case len(matches) < 1:
		return util.NotFoundError.Errorf("block data, %q(%d) not found", dataType, height)
	case len(matches) > 1:
		return xerrors.Errorf("block data, %q(%d) multiple files found", dataType, height)
	default:
		f = matches[0]
	}

	_, _, checksum, err := localfs.ParseDataFileName(f)
	if err != nil {
		return err
	}

	if i, err := util.GenerateFileChecksum(f); err != nil {
		return err
	} else if checksum != i {
		return xerrors.Errorf("file checksum does not match; %s != %s", checksum, i)
	}

	return nil
}
