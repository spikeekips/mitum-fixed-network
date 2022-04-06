package cmds

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	ContextValueWhenBlockSaved util.ContextKey = "restore_when_block_saved"
	ContextValueWhenFinished   util.ContextKey = "restore_when_finished"
)

type RestoreCommand struct {
	*BaseRunCommand
	Clean                 bool   `help:"clean the existing data"`
	Concurrency           uint64 `help:"how many blocks are handled at same time default: 10" default:"10"`
	Dryrun                bool   `help:"just check blockdata and database default: false" default:"false"`
	One                   string `help:"restore one blockdata"`
	enc                   *jsonenc.Encoder
	database              storage.Database
	blockdata             *localfs.Blockdata
	networkID             base.NetworkID
	from                  base.Height
	to                    base.Height
	whenBlockSaved        func(block.Block) error
	whenFinished          func(base.Height) error
	cleanDatabase         func() error
	cleanDatabaseByHeight func(context.Context, base.Height) error
	oneHeight             base.Height
}

func NewRestoreCommand() RestoreCommand {
	cmd := RestoreCommand{
		BaseRunCommand: NewBaseRunCommand(false, "restore-blockdata"),
		to:             base.NilHeight,
	}

	ps := cmd.Processes()
	if ps == nil {
		panic(errors.Errorf("processes not prepared"))
	}

	for _, i := range []pm.Process{
		process.ProcessorConsensusStates,
		process.ProcessorNetwork,
		process.ProcessorProposalProcessor,
	} {
		if err := ps.AddProcess(pm.NewDisabledProcess(i), true); err != nil {
			panic(err)
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *RestoreCommand) Run(version util.Version) error {
	s := time.Now()

	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}
	defer cmd.Done()
	defer func() {
		<-time.After(time.Second * 1)
	}()

	cmd.Log().Info().
		Bool("dryrun", cmd.Dryrun).
		Uint64("concurrency", cmd.Concurrency).
		Str("one", cmd.One).
		Msg("started")

	if err := cmd.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()
	_ = ps.SetContext(context.WithValue(ps.ContextSource(), process.ContextValueGenesisBlockForceCreate, false))
	_ = cmd.SetProcesses(ps)

	if err := ps.Run(); err != nil {
		return err
	}

	if err := cmd.load(ps.Context()); err != nil {
		return err
	}

	if err := cmd.checkBlockdata(); err != nil {
		return err
	}

	if cmd.Dryrun {
		return nil
	}

	if err := cmd.restoreBlockdata(); err != nil {
		return err
	}

	cmd.Log().Info().Dur("elapsed", time.Since(s)).Msg("all blockdata restored")

	return nil
}

func (cmd *RestoreCommand) prepare() error {
	if err := cmd.BaseRunCommand.prepare(); err != nil {
		return err
	}

	if len(cmd.One) > 0 {
		switch fi, err := os.Stat(cmd.One); {
		case err != nil:
			if os.IsNotExist(err) {
				return fmt.Errorf("blockdata, %q does not exist: %w", cmd.One, err)
			}

			return fmt.Errorf("failed to access blockdata, %q: %w", cmd.One, err)
		case !fi.IsDir():
			return fmt.Errorf("blockdata, %q is not directory", cmd.One)
		}
	}

	return nil
}

func (cmd *RestoreCommand) load(ctx context.Context) error {
	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return err
	}
	cmd.networkID = policy.NetworkID()

	if err := config.LoadJSONEncoderContextValue(ctx, &cmd.enc); err != nil {
		return err
	}

	if err := process.LoadDatabaseContextValue(ctx, &cmd.database); err != nil {
		return err
	}

	if err := util.LoadFromContextValue(ctx, process.ContextValueBlockdata, &cmd.blockdata); err != nil {
		return err
	}

	if err := util.LoadFromContextValue(ctx, ContextValueWhenBlockSaved, &cmd.whenBlockSaved); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return err
		}
	}

	if err := util.LoadFromContextValue(ctx, ContextValueWhenFinished, &cmd.whenFinished); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return err
		}
	}

	if err := util.LoadFromContextValue(ctx, ContextValueCleanDatabase, &cmd.cleanDatabase); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return err
		}
	}

	if err := util.LoadFromContextValue(ctx, ContextValueCleanDatabaseByHeight, &cmd.cleanDatabaseByHeight); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return err
		}
	}

	return nil
}

func (cmd *RestoreCommand) checkBlockdata() error {
	s := time.Now()

	if len(cmd.One) > 0 {
		return cmd.checkOneBlockdata()
	}

	if err := cmd.checkBlockdataPath(); err != nil {
		return err
	}

	cmd.Log().Info().Dur("elapsed", time.Since(s)).Msg("blockdata checked")

	return nil
}

func (cmd *RestoreCommand) checkOneBlockdata() error {
	s := time.Now()
	cmd.Log().Debug().Msg("trying to check one blockdata")

	_, blk, err := localfs.LoadBlockByPath(cmd.blockdata, cmd.One)
	if err != nil {
		return err
	}

	if !cmd.Clean {
		switch _, found, err := cmd.database.ManifestByHeight(blk.Height()); { // nolint:govet
		case err != nil:
			return fmt.Errorf("failed to get block, %d: %w", blk.Height(), err)
		case found:
			return fmt.Errorf("block, %d found; use --clean", blk.Height())
		}
	}

	prev, found, err := cmd.database.ManifestByHeight(blk.Height() - 1)
	switch {
	case err != nil:
		return fmt.Errorf("failed to get previous block, %d: %w", blk.Height()-1, err)
	case !found:
		return fmt.Errorf("previous block, %d not found", blk.Height()-1)
	}

	if cmd.Dryrun {
		l := strings.Repeat("-", 80)
		_, _ = fmt.Fprintf(os.Stdout, `%s
* last in database: %s(%s)
*        blockdata: %s(%s)
%s
`, l, prev.Height().String(), prev.Hash().String(), blk.Height().String(), blk.Hash().String(), l)
	}

	if err = blk.IsValid(cmd.networkID); err != nil {
		return err
	}

	checker := isaac.NewManifestsValidationChecker(cmd.networkID, []block.Manifest{prev, blk.Manifest()})
	_ = checker.SetLogging(cmd.Logging)

	if err := util.NewChecker("manifests-validation-checker", []util.CheckerFunc{
		checker.CheckSerialized,
	}).Check(); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to verify manifests")

		return err
	}

	cmd.Log().Debug().Dur("elapsed", time.Since(s)).Msg("checked blockdata")

	cmd.oneHeight = blk.Height()

	return nil
}

func (cmd *RestoreCommand) checkBlockdataPath() error {
	lastHeight := base.NilHeight
	var lastHash valuehash.Hash

	from := base.PreGenesisHeight

	if !cmd.Clean {
		i, j, err := cmd.checkLastBlockInDatabase()
		if err != nil {
			return err
		}

		lastHeight = i
		lastHash = j
		from = i + 1
	}

	to, err := cmd.checkBlockdataExists(from)
	if err != nil {
		return err
	}

	switch {
	case to <= base.PreGenesisHeight:
		return errors.Errorf("blockdata; nothing to restore")
	case !cmd.Clean && from > to:
		return errors.Errorf("already restored; %d > %d, use --clean", from, to)
	}

	if cmd.Dryrun {
		last := "not found"
		if lastHeight > base.NilHeight && lastHash != nil {
			last = fmt.Sprintf("%s(%s)", lastHeight.String(), lastHash.String())
		}

		l := strings.Repeat("-", 80)
		_, _ = fmt.Fprintf(os.Stdout, `%s
* in database:
  last: %s
* in blockdata:
  from: %d
    to: %d
%s
`, l, last, from, to, l)
	}

	c := int64(cmd.Concurrency)
	d := int64(math.Ceil(float64((to - from).Int64()) / float64(c)))
	for i := int64(0); i < d; i++ {
		s := from.Int64() + (i * c)
		e := s + c
		if t := to.Int64(); e > t {
			e = t
		}

		if err := cmd.checkBlockdatas(base.Height(s), base.Height(e)); err != nil {
			return err
		}
	}

	cmd.Log().Debug().Interface("from_to", []int64{from.Int64(), to.Int64()}).Msg("heights found")

	cmd.from = from
	cmd.to = to

	return nil
}

func (cmd *RestoreCommand) checkLastBlockInDatabase() (base.Height, valuehash.Hash, error) {
	last := base.NilHeight
	var lastHash valuehash.Hash

	// NOTE check last manifest from database
	switch blk, found, err := cmd.database.LastManifest(); {
	case err != nil:
		return last, nil, fmt.Errorf("failed to get last manifest from database info: %w", err)
	case found:
		return blk.Height(), blk.Hash(), nil
	}

	if err := cmd.database.Manifests(false, true, 1,
		func(height base.Height, h valuehash.Hash, _ block.Manifest) (bool, error) {
			last = height
			lastHash = h

			return false, nil
		},
	); err != nil {
		return last, nil, fmt.Errorf("failed to get last manifest from database: %w", err)
	}

	return last, lastHash, nil
}

func (cmd *RestoreCommand) checkBlockdataExists(from base.Height) (base.Height, error) {
	height := from
	to := base.NilHeight

end:
	for {
		switch found, removed, err := cmd.blockdata.ExistsReal(height); {
		case err != nil:
			return to, fmt.Errorf("failed to check blockdata, %d: %w", height, err)
		case !found:
			break end
		case removed:
			return to, fmt.Errorf("blockdata, %d found, but removed", height)
		}

		to = height
		height++
	}

	return to, nil
}

func (cmd *RestoreCommand) checkBlockdatas(from, to base.Height) error {
	wk := util.NewErrgroupWorker(context.Background(), int64(cmd.Concurrency))
	defer wk.Close()

	s := time.Now()
	l := cmd.Log().With().Interface("from_to", []base.Height{from, to}).Logger()
	l.Debug().Msg("trying to check blockdata")

	manifests := make([]block.Manifest, (to + 1 - from).Int64())
	go func() {
		defer wk.Done()

		for height := from; height < to+1; height++ {
			h := height
			if err := wk.NewJob(func(ctx context.Context, _ uint64) error {
				blk, err := cmd.checkBlockdataByHeight(h)
				if err != nil {
					return err
				}

				manifests[blk.Height()-from] = blk.Manifest()

				return nil
			}); err != nil {
				l.Error().Err(err).Int64("height", h.Int64()).Msg("failed to NewJob for checking blockdata")
			}
		}
	}()

	if err := wk.Wait(); err != nil {
		return fmt.Errorf("failed to check blockdata: %w", err)
	}

	checker := isaac.NewManifestsValidationChecker(cmd.networkID, manifests)
	_ = checker.SetLogging(cmd.Logging)

	if err := util.NewChecker("manifests-validation-checker", []util.CheckerFunc{
		checker.CheckSerialized,
	}).Check(); err != nil {
		l.Error().Err(err).Msg("failed to verify manifests")

		return err
	}

	l.Debug().Dur("elapsed", time.Since(s)).Msg("checked blockdata")

	return nil
}

func (cmd *RestoreCommand) checkBlockdataByHeight(height base.Height) (block.Block, error) {
	_, blk, err := localfs.LoadBlock(cmd.blockdata, height)
	if err != nil {
		return nil, err
	}

	if err := blk.IsValid(cmd.networkID); err != nil {
		return nil, err
	}

	return blk, nil
}

func (cmd *RestoreCommand) restoreBlockdata() error {
	if len(cmd.One) > 0 {
		if err := cmd.restoreOneBlockdata(); err != nil {
			return err
		}
	} else if err := cmd.restoreBlockdataPath(); err != nil {
		return err
	}

	if cmd.whenFinished != nil {
		if err := cmd.whenFinished(cmd.to); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *RestoreCommand) restoreOneBlockdata() error {
	if cmd.Clean {
		if err := cmd.database.CleanByHeight(cmd.oneHeight); err != nil {
			return err
		}

		if cmd.cleanDatabaseByHeight != nil {
			if err := cmd.cleanDatabaseByHeight(context.Background(), cmd.oneHeight); err != nil {
				return err
			}
		}

		height := cmd.oneHeight

	end:
		for {
			switch found, err := cmd.blockdata.Exists(height); {
			case err != nil:
				return err
			case !found:
				break end
			}

			if err := cmd.blockdata.RemoveAll(height); err != nil {
				return err
			}
			height++
		}
	}

	// NOTE copy blockdata files under blockdata
	dst := filepath.Join(cmd.blockdata.Root(), localfs.HeightDirectory(cmd.oneHeight))
	if err := copyBlockdataDirectory(cmd.One, dst); err != nil {
		return fmt.Errorf("failed to copy blockdata files: %w", err)
	}

	bdm, blk, err := localfs.LoadBlockByPath(cmd.blockdata, dst)
	if err != nil {
		cmd.Log().Error().Str("path", cmd.One).Err(err).Msg("failed to load block")

		return err
	}

	return cmd.saveBlockdata(bdm, blk)
}

func (cmd *RestoreCommand) restoreBlockdataPath() error {
	if cmd.Clean {
		if err := cmd.database.Clean(); err != nil {
			return err
		}

		if cmd.cleanDatabase != nil {
			if err := cmd.cleanDatabase(); err != nil {
				return err
			}
		}
	}

	wk := util.NewErrgroupWorker(context.Background(), int64(cmd.Concurrency))
	defer wk.Close()

	errch := make(chan error, 2)
	go func() {
		defer wk.Done()

		for i := cmd.from; i < cmd.to+1; i++ {
			height := i
			err := wk.NewJob(func(_ context.Context, _ uint64) error {
				return cmd.restoreBlockdataByHeight(height)
			})
			if err != nil {
				cmd.Log().Error().Err(err).Msg("failed to NewJob for restore blockdata")

				errch <- err

				return
			}
		}

		errch <- nil
	}()

	if err := wk.Wait(); err != nil {
		return fmt.Errorf("failed to restore blockdata: %w", err)
	}

	return <-errch
}

func (cmd *RestoreCommand) restoreBlockdataByHeight(height base.Height) error {
	bdm, blk, err := localfs.LoadBlock(cmd.blockdata, height)
	if err != nil {
		cmd.Log().Error().Int64("height", height.Int64()).Err(err).Msg("failed to load block")

		return err
	}

	return cmd.saveBlockdata(bdm, blk)
}

func (cmd *RestoreCommand) saveBlockdata(bdm block.BaseBlockdataMap, blk block.Block) error {
	s := time.Now()

	sst, err := cmd.database.NewSyncerSession()
	if err != nil {
		return err
	}

	defer func() {
		_ = sst.Close()
	}()

	height := blk.Height()
	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	if cmd.to > base.NilHeight && height != cmd.to {
		sst.SetSkipLastBlock(true)
	}

	if err := sst.SetBlocks([]block.Block{blk}, []block.BlockdataMap{bdm}); err != nil {
		return err
	}

	if err := sst.Commit(); err != nil {
		return err
	}

	if cmd.to == base.NilHeight || height == cmd.to {
		if db, ok := cmd.database.(storage.LastBlockSaver); ok {
			if err := db.SaveLastBlock(height); err != nil {
				return err
			}
		}
	}

	if cmd.whenBlockSaved != nil {
		if err := cmd.whenBlockSaved(blk); err != nil {
			return err
		}
	}

	l.Debug().Dur("elapsed", time.Since(s)).Msg("blockdata restored")

	return nil
}

func copyBlockdataDirectory(src, dst string) error {
	srca, err := checkBlockdataDirectory(src)
	if err != nil {
		return fmt.Errorf("invalid source directory, %q: %w", src, err)
	}

	dsta, err := checkBlockdataDirectory(dst)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("invalid destination directory, %q: %w", dst, err)
		}
	}

	if err := os.MkdirAll(dsta, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory, %q: %w", dsta, err)
	}

	for i := range block.Blockdata {
		d := block.Blockdata[i]
		if err := copyBlockdataFile(srca, dst, d); err != nil {
			return fmt.Errorf("failed to copy blockdata file, %q: %w", d, err)
		}
	}

	return nil
}

func checkBlockdataDirectory(d string) (string, error) {
	da, err := filepath.Abs(d)
	if err != nil {
		return "", fmt.Errorf("invalid directory, %q: %w", d, err)
	}

	switch fi, err := os.Stat(da); {
	case err != nil:
		if os.IsNotExist(err) {
			return da, fmt.Errorf("directory, %q does not exist: %w", da, err)
		}

		return da, fmt.Errorf("failed to access directory, %q: %w", da, err)
	case !fi.IsDir():
		return da, fmt.Errorf("%q is not directory", da)
	}

	return da, nil
}

func copyBlockdataFile(src, dst, datatype string) error {
	f, source, err := localfs.OpenFile(src, datatype)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

	nf := filepath.Join(dst, filepath.Base(f))

	destination, err := os.Create(filepath.Clean(nf))
	if err != nil {
		return err
	}

	_, err = io.Copy(destination, source)
	_ = destination.Close()

	return err
}
