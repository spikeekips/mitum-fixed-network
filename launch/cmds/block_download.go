package cmds

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"

	"github.com/alecthomas/kong"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

var allBlockData = "all"

var BlockDownloadVars = kong.Vars{
	"block_datatypes": strings.Join(block.BlockData, ","),
	"all_blockdata":   allBlockData,
}

type BlockDownloadCommand struct {
	*BaseCommand
	DataType   string        `arg:"" name:"data type" help:"data type of block data, {${block_datatypes} ${all_blockdata}}" required:""` // nolint:lll
	Heights    []int64       `arg:"" name:"height" help:"block heights of block" required:""`
	URL        *url.URL      `name:"node" help:"remote mitum url. default: ${node_url}" required:"" default:"${node_url}"`
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	Save       string        `name:"save" help:"save block data under directory"`
	channel    network.Channel
	heights    []base.Height
	bd         blockdata.BlockData
}

func NewBlockDownloadCommand() BlockDownloadCommand {
	return BlockDownloadCommand{
		BaseCommand: NewBaseCommand("block-download"),
	}
}

func (cmd *BlockDownloadCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL.String()).Ints64("heights", cmd.Heights).Msg("trying to get block")

	if err := cmd.prepare(); err != nil {
		return err
	}

	switch cmd.DataType {
	case "map":
		if err := cmd.printBlockDataMaps(cmd.heights); err != nil {
			return xerrors.Errorf("failed to get block data maps: %w", err)
		}
	default:
		if err := cmd.blockData(cmd.heights); err != nil {
			return xerrors.Errorf("failed to get block data: %w", err)
		}
	}

	return nil
}

func (cmd *BlockDownloadCommand) prepare() error {
	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	if err := cmd.prepareHeight(); err != nil {
		return err
	}
	if err := cmd.prepareChannel(); err != nil {
		return err
	}
	if err := cmd.prepareDataType(); err != nil {
		return err
	}
	if err := cmd.prepareBlockData(); err != nil {
		return err
	}

	return nil
}

func (cmd *BlockDownloadCommand) prepareHeight() error {
	m := map[base.Height]struct{}{}
	heights := make([]base.Height, len(cmd.Heights))
	for i := range cmd.Heights {
		h := base.Height(cmd.Heights[i])
		if err := h.IsValid(nil); err != nil {
			return err
		} else if _, found := m[h]; found {
			return xerrors.Errorf("duplicated height, %d", h)
		} else {
			m[h] = struct{}{}
			heights[i] = h
		}
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i]-heights[j] < 0
	})
	cmd.heights = heights

	return nil
}

func (cmd *BlockDownloadCommand) prepareChannel() error {
	encs := cmd.Encoders()
	if encs == nil {
		if i, err := cmd.LoadEncoders(nil); err != nil {
			return err
		} else {
			encs = i
		}
	}

	if cmd.TLSInscure {
		query := cmd.URL.Query()
		query.Set("insecure", "true")
		cmd.URL.RawQuery = query.Encode()
	}

	if ch, err := process.LoadNodeChannel(cmd.URL, encs, cmd.Timeout); err != nil {
		return err
	} else {
		cmd.channel = ch
	}

	cmd.Log().Debug().Msg("network channel loaded")

	return nil
}

func (cmd *BlockDownloadCommand) prepareDataType() error {
	switch d := cmd.DataType; d {
	case "map":
	case allBlockData:
	default:
		var found bool
		for i := range block.BlockData {
			if d == block.BlockData[i] {
				found = true

				break
			}
		}

		if !found {
			return xerrors.Errorf("unknown block data type, %q", d)
		}
	}

	return nil
}

func (cmd *BlockDownloadCommand) prepareBlockData() error {
	cmd.Save = strings.TrimSpace(cmd.Save)
	if len(cmd.Save) < 1 {
		return nil
	}

	if i, err := os.Stat(filepath.Clean(cmd.Save)); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(cmd.Save, localfs.DefaultDirectoryPermission); err != nil {
			return err
		}
	} else if !i.IsDir() {
		return xerrors.Errorf("save path, %q is not directory", cmd.Save)
	}

	cmd.bd = localfs.NewBlockData(cmd.Save, cmd.jsonenc)
	if err := cmd.bd.Initialize(); err != nil {
		return err
	}

	return nil
}

func (cmd *BlockDownloadCommand) blockDataMaps(heights []base.Height) ([]block.BlockDataMap, error) {
	ch := make(chan []block.BlockDataMap)
	errch := make(chan error)

	go func() {
		defer func() {
			close(ch)
		}()

		err := requestBlockDataMaps(heights, func(hs []base.Height) error {
			ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
			defer cancel()

			if m, err := cmd.channel.BlockDataMaps(ctx, hs); err != nil {
				return err
			} else {
				sort.Slice(m, func(i, j int) bool {
					return m[i].Height()-m[j].Height() < 0
				})

				ch <- m

				return nil
			}
		})
		if err != nil {
			errch <- err
		}
	}()

	var maps []block.BlockDataMap

end:
	for {
		select {
		case err := <-errch:
			cmd.Log().Error().Err(err).Interface("heights", heights).Msg("failed to request block data maps")

			return nil, err
		case i, notclosed := <-ch:
			if !notclosed {
				break end
			}

			maps = append(maps, i...)
		}
	}

	cmd.Log().Debug().Interface("heights", heights).Msg("block data maps")

	return maps, nil
}

func (cmd *BlockDownloadCommand) blockData(heights []base.Height) error {
	var maps []block.BlockDataMap
	if i, err := cmd.blockDataMaps(heights); err != nil {
		return xerrors.Errorf("failed to get block data maps: %w", err)
	} else {
		maps = i
	}

	return requestBlockData(maps, func(m block.BlockDataMap) error {
		if err := cmd.oneBlockData(m); err != nil {
			return xerrors.Errorf("failed to get one block data, %d: %w", m.Height(), err)
		} else {
			return nil
		}
	})
}

func (cmd *BlockDownloadCommand) printBlockDataMaps(heights []base.Height) error {
	var maps []block.BlockDataMap
	if i, err := cmd.blockDataMaps(heights); err != nil {
		return err
	} else {
		maps = i
	}

	cmd.Log().Debug().Msg("block data maps thru channel")
	for i := range maps {
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(maps[i]))
	}

	return nil
}

func (cmd *BlockDownloadCommand) oneBlockData(m block.BlockDataMap) error {
	var items []block.BlockDataMapItem
	switch cmd.DataType {
	case block.BlockDataManifest,
		block.BlockDataOperations,
		block.BlockDataOperationsTree,
		block.BlockDataStates,
		block.BlockDataStatesTree,
		block.BlockDataINITVoteproof,
		block.BlockDataACCEPTVoteproof,
		block.BlockDataSuffrageInfo,
		block.BlockDataProposal:
		if i, err := getItemBlockDataMap(m, cmd.DataType); err != nil {
			return err
		} else {
			items = append(items, i)
		}
	case allBlockData:
		items = make([]block.BlockDataMapItem, len(block.BlockData))
		for i := range block.BlockData {
			if j, err := getItemBlockDataMap(m, block.BlockData[i]); err != nil {
				return err
			} else {
				items[i] = j
			}
		}
	default:
		return xerrors.Errorf("unknown data type found, %q", cmd.DataType)
	}

	if cmd.bd == nil {
		return cmd.printBlockData(m, items)
	} else if err := cmd.saveBlockData(m, items); err != nil {
		return err
	}

	return nil
}

func (cmd *BlockDownloadCommand) printBlockData(m block.BlockDataMap, items []block.BlockDataMapItem) error {
	for i := range items {
		item := items[i]

		s := fmt.Sprintf("{\"height\": %d, \"data_type\": %q}\n", m.Height(), item.Type())
		if j, err := cmd.printBlockDataItem(item); err != nil {
			return err
		} else if b, err := jsonenc.MarshalIndent(j); err != nil {
			return err
		} else {
			s += string(b)
		}
		_, _ = fmt.Fprintln(os.Stdout, s)
	}

	return nil
}

func (cmd *BlockDownloadCommand) printBlockDataItem(item block.BlockDataMapItem) (interface{}, error) {
	var r io.ReadCloser
	if i, err := cmd.requestChannel(item); err != nil {
		return nil, err
	} else {
		r = i
	}

	writer := blockdata.NewDefaultWriter(cmd.jsonenc)

	switch item.Type() {
	case block.BlockDataManifest:
		return writer.ReadManifest(r)
	case block.BlockDataOperations:
		return writer.ReadOperations(r)
	case block.BlockDataOperationsTree:
		return writer.ReadOperationsTree(r)
	case block.BlockDataStates:
		return writer.ReadStates(r)
	case block.BlockDataStatesTree:
		return writer.ReadStatesTree(r)
	case block.BlockDataINITVoteproof:
		return writer.ReadINITVoteproof(r)
	case block.BlockDataACCEPTVoteproof:
		return writer.ReadACCEPTVoteproof(r)
	case block.BlockDataSuffrageInfo:
		return writer.ReadSuffrageInfo(r)
	case block.BlockDataProposal:
		return writer.ReadProposal(r)
	default:
		return nil, xerrors.Errorf("unknown data type found, %q", item.Type())
	}
}

func (cmd *BlockDownloadCommand) saveBlockData(m block.BlockDataMap, items []block.BlockDataMapItem) error {
	var session blockdata.Session
	if i, err := cmd.bd.NewSession(m.Height()); err != nil {
		return err
	} else {
		defer func() {
			_ = i.Cancel()
		}()

		session = i
	}

	base := filepath.Join(cmd.Save, localfs.HeightDirectory(m.Height()))
	if i, err := os.Stat(base); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(base, localfs.DefaultDirectoryPermission); err != nil {
			return err
		}
	} else if !i.IsDir() {
		return xerrors.Errorf("block directory, %q already exists", base)
	}

	for i := range items {
		if err := cmd.saveBlockDataItem(m, items[i], session); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *BlockDownloadCommand) saveBlockDataItem(
	m block.BlockDataMap,
	item block.BlockDataMapItem,
	session blockdata.Session,
) error {
	var r io.ReadCloser
	if i, err := cmd.requestChannel(item); err != nil {
		return err
	} else {
		r = i
	}

	if i, err := session.Import(item.Type(), r); err != nil {
		return xerrors.Errorf("failed to import block data: %w", err)
	} else {
		base := filepath.Join(cmd.Save, localfs.HeightDirectory(m.Height()))
		f := filepath.Join(base, filepath.Base(i))
		if err := os.Rename(i, f); err != nil {
			return err
		}

		cmd.Log().Info().Int64("height", m.Height().Int64()).Str("file", f).Msg("saved")

		return nil
	}
}

func (cmd *BlockDownloadCommand) requestChannel(item block.BlockDataMapItem) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	return cmd.channel.BlockData(ctx, item)
}

func getItemBlockDataMap(m block.BlockDataMap, dataType string) (block.BlockDataMapItem, error) {
	var item block.BlockDataMapItem
	switch dataType {
	case block.BlockDataManifest:
		item = m.Manifest()
	case block.BlockDataOperations:
		item = m.Operations()
	case block.BlockDataOperationsTree:
		item = m.OperationsTree()
	case block.BlockDataStates:
		item = m.States()
	case block.BlockDataStatesTree:
		item = m.StatesTree()
	case block.BlockDataINITVoteproof:
		item = m.INITVoteproof()
	case block.BlockDataACCEPTVoteproof:
		item = m.ACCEPTVoteproof()
	case block.BlockDataSuffrageInfo:
		item = m.SuffrageInfo()
	case block.BlockDataProposal:
		item = m.Proposal()
	default:
		return nil, xerrors.Errorf("unknown data type found, %q", dataType)
	}

	return item, nil
}

func requestBlockDataMaps(heights []base.Height, callback func([]base.Height) error) error {
	sem := semaphore.NewWeighted(int64(quicnetwork.LimitRequestByHeights))
	eg, ctx := errgroup.WithContext(context.Background())

	limit := quicnetwork.LimitRequestByHeights
	l := len(heights) / limit
	if len(heights)%limit != 0 {
		l++
	}

	for i := 0; i < l; i++ {
		e := (i + 1) * limit
		if e > len(heights) {
			e = len(heights)
		}
		hs := heights[i*limit : e]
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		eg.Go(func() error {
			defer sem.Release(1)

			return callback(hs)
		})
	}

	if err := sem.Acquire(ctx, 10); err != nil {
		return err
	} else if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func requestBlockData(maps []block.BlockDataMap, callback func(block.BlockDataMap) error) error {
	sem := semaphore.NewWeighted(10)
	eg, ctx := errgroup.WithContext(context.Background())

	for i := range maps {
		m := maps[i]
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		eg.Go(func() error {
			defer sem.Release(1)

			return callback(m)
		})
	}

	if err := sem.Acquire(ctx, 10); err != nil {
		return err
	} else if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
