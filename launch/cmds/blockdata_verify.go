package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

type BlockdataVerifyCommand struct {
	*BaseVerifyCommand
	Path string `arg:"" name:"blockdata path"`
	bd   blockdata.Blockdata
}

func NewBlockdataVerifyCommand(types []hint.Type, hinters []hint.Hinter) BlockdataVerifyCommand {
	return BlockdataVerifyCommand{
		BaseVerifyCommand: NewBaseVerifyCommand("blockdata-verify", types, hinters),
	}
}

func (cmd *BlockdataVerifyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	cmd.Log().Debug().Str("path", cmd.Path).Msg("trying to verify blockdata")

	return cmd.verify()
}

func (cmd *BlockdataVerifyCommand) Initialize(flags interface{}, version util.Version) error {
	if err := cmd.BaseVerifyCommand.Initialize(flags, version); err != nil {
		return err
	}

	if i, err := os.Stat(cmd.Path); err != nil {
		return errors.Wrapf(err, "invalid path, %q", cmd.Path)
	} else if !i.IsDir() {
		return errors.Errorf("path, %q is not directory", cmd.Path)
	}

	cmd.bd = localfs.NewBlockdata(cmd.Path, cmd.jsonenc)
	return cmd.bd.Initialize()
}

func (cmd *BlockdataVerifyCommand) verify() error {
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

func (cmd *BlockdataVerifyCommand) checkLastHeight() error {
	var height base.Height = base.PreGenesisHeight
	for {
		if found, err := cmd.bd.Exists(height); err != nil {
			return errors.Wrapf(err, "failed to check blockdata of height, %d", height)
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

func (cmd *BlockdataVerifyCommand) loadManifest(height base.Height) (block.Manifest, error) {
	bd := cmd.bd.(*localfs.Blockdata)

	prepath := filepath.Join(bd.Root(), localfs.HeightDirectory(height))
	var manifest block.Manifest
	_, i, err := localfs.LoadData(prepath, block.BlockdataManifest)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = i.Close()
	}()

	if j, err := cmd.bd.Writer().ReadManifest(i); err != nil {
		return nil, err
	} else if err := j.IsValid(cmd.networkID); err != nil {
		return nil, errors.Wrapf(err, "invalid manifest, %q found", height)
	} else {
		manifest = j
	}

	return manifest, nil
}

func (cmd *BlockdataVerifyCommand) checkBlocks() error {
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := base.PreGenesisHeight; i <= cmd.lastHeight; i++ {
			height := i

			if err := wk.NewJob(func(context.Context, uint64) error {
				_, err := cmd.loadBlock(height)

				return err
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}

func (cmd *BlockdataVerifyCommand) loadBlock(height base.Height) (block.Block, error) {
	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	if _, i, err := localfs.LoadBlock(cmd.bd.(*localfs.Blockdata), height); err != nil {
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

func (cmd *BlockdataVerifyCommand) checkAllBlockFiles() error {
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := base.PreGenesisHeight; i <= cmd.lastHeight; i++ {
			height := i

			if err := wk.NewJob(func(context.Context, uint64) error {
				return cmd.checkBlockFiles(height)
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}

func (cmd *BlockdataVerifyCommand) checkBlockFiles(height base.Height) error {
	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	if found, err := cmd.bd.Exists(height); err != nil {
		return err
	} else if !found {
		return util.NotFoundError.Errorf("block data %d not found", height)
	}

	var hasError bool
	for i := range block.Blockdata {
		dataType := block.Blockdata[i]
		if err := cmd.checkBlockFile(height, dataType); err != nil {
			l.Error().Err(err).
				Int64("height", height.Int64()).
				Str("data_type", dataType).
				Msg("failed to check block data file")

			hasError = true
		}
	}

	if hasError {
		return errors.Errorf("block data file of height, %d has problem", height)
	}
	l.Debug().Msg("block data files checked")

	return nil
}

func (cmd *BlockdataVerifyCommand) checkBlockFile(height base.Height, dataType string) error {
	g := filepath.Join(cmd.Path, localfs.HeightDirectory(height), fmt.Sprintf("%d-%s-*.jsonld.gz", height, dataType))

	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return storage.MergeStorageError(err)
	case len(matches) < 1:
		return util.NotFoundError.Errorf("block data, %q(%d) not found", dataType, height)
	case len(matches) > 1:
		return errors.Errorf("block data, %q(%d) multiple files found", dataType, height)
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
		return errors.Errorf("file checksum does not match; %s != %s", checksum, i)
	}

	return nil
}
