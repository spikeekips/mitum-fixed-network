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

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var allBlockdata = "all"

var BlockDownloadVars = kong.Vars{
	"block_datatypes": strings.Join(block.Blockdata, ","),
	"all_blockdata":   allBlockdata,
}

type BlockDownloadCommand struct {
	*BaseCommand
	DataType   string        `arg:"" name:"data type" help:"data type of block data, {${block_datatypes} ${all_blockdata}}" required:"true"` // revive:disable-line:line-length-limit
	Heights    []int64       `arg:"" name:"height" help:"heights of block" required:"true"`
	URL        *url.URL      `name:"node" help:"remote mitum url. default: ${node_url}" required:"true" default:"${node_url}"` // revive:disable-line:line-length-limit
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	Save       string        `name:"save" help:"save block data under directory"`
	channel    network.Channel
	heights    []base.Height
	bd         blockdata.Blockdata
}

func NewBlockDownloadCommand(types []hint.Type, hinters []hint.Hinter) BlockDownloadCommand {
	cmd := BlockDownloadCommand{
		BaseCommand: NewBaseCommand("block-download"),
	}

	if _, err := cmd.LoadEncoders(types, hinters); err != nil {
		panic(err)
	}

	return cmd
}

func (cmd *BlockDownloadCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL.String()).Ints64("heights", cmd.Heights).Msg("trying to get block")

	if err := cmd.prepare(); err != nil {
		return err
	}

	switch cmd.DataType {
	case "map":
		if err := cmd.printBlockdataMaps(cmd.heights); err != nil {
			return errors.Wrap(err, "failed to get block data maps")
		}
	default:
		if err := cmd.getBlockdata(cmd.heights); err != nil {
			return errors.Wrap(err, "failed to get block data")
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
	return cmd.prepareBlockdata()
}

func (cmd *BlockDownloadCommand) prepareHeight() error {
	m := map[base.Height]struct{}{}
	heights := make([]base.Height, len(cmd.Heights))
	for i := range cmd.Heights {
		h := base.Height(cmd.Heights[i])
		if err := h.IsValid(nil); err != nil {
			return err
		} else if _, found := m[h]; found {
			return errors.Errorf("duplicated height, %d", h)
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
		i, err := cmd.LoadEncoders(nil, nil)
		if err != nil {
			return err
		}
		encs = i
	}

	connInfo := network.NewHTTPConnInfo(network.NormalizeURL(cmd.URL), cmd.TLSInscure)
	ch, err := process.LoadNodeChannel(connInfo, encs, cmd.Timeout)
	if err != nil {
		return err
	}
	cmd.channel = ch

	cmd.Log().Debug().Msg("network channel loaded")

	return nil
}

func (cmd *BlockDownloadCommand) prepareDataType() error {
	switch d := cmd.DataType; d {
	case "map":
	case allBlockdata:
	default:
		var found bool
		for i := range block.Blockdata {
			if d == block.Blockdata[i] {
				found = true

				break
			}
		}

		if !found {
			return errors.Errorf("unknown block data type, %q", d)
		}
	}

	return nil
}

func (cmd *BlockDownloadCommand) prepareBlockdata() error {
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
		return errors.Errorf("save path, %q is not directory", cmd.Save)
	}

	cmd.bd = localfs.NewBlockdata(cmd.Save, cmd.jsonenc)
	return cmd.bd.Initialize()
}

func (cmd *BlockDownloadCommand) blockdataMaps(heights []base.Height) ([]block.BlockdataMap, error) {
	ch := make(chan []block.BlockdataMap)
	errch := make(chan error)

	go func() {
		defer func() {
			close(ch)
		}()

		err := requestBlockdataMaps(heights, func(hs []base.Height) error {
			ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
			defer cancel()

			m, err := cmd.channel.BlockdataMaps(ctx, hs)
			if err != nil {
				return err
			}
			sort.Slice(m, func(i, j int) bool {
				return m[i].Height()-m[j].Height() < 0
			})

			ch <- m

			return nil
		})
		if err != nil {
			errch <- err
		}
	}()

	var maps []block.BlockdataMap

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

func (cmd *BlockDownloadCommand) getBlockdata(heights []base.Height) error {
	maps, err := cmd.blockdataMaps(heights)
	if err != nil {
		return errors.Wrap(err, "failed to get block data maps")
	}

	return requestBlockdata(maps, func(m block.BlockdataMap) error {
		if err := cmd.oneBlockdata(m); err != nil {
			return errors.Wrapf(err, "failed to get one block data, %d", m.Height())
		}
		return nil
	})
}

func (cmd *BlockDownloadCommand) printBlockdataMaps(heights []base.Height) error {
	maps, err := cmd.blockdataMaps(heights)
	if err != nil {
		return err
	}

	cmd.Log().Debug().Msg("block data maps thru channel")
	for i := range maps {
		b, err := cmd.jsonenc.Marshal(maps[i])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, string(b))
	}

	return nil
}

func (cmd *BlockDownloadCommand) oneBlockdata(m block.BlockdataMap) error {
	var items []block.BlockdataMapItem
	switch cmd.DataType {
	case block.BlockdataManifest,
		block.BlockdataOperations,
		block.BlockdataOperationsTree,
		block.BlockdataStates,
		block.BlockdataStatesTree,
		block.BlockdataINITVoteproof,
		block.BlockdataACCEPTVoteproof,
		block.BlockdataSuffrageInfo,
		block.BlockdataProposal:
		i, err := getItemBlockdataMap(m, cmd.DataType)
		if err != nil {
			return err
		}
		items = append(items, i)
	case allBlockdata:
		items = make([]block.BlockdataMapItem, len(block.Blockdata))
		for i := range block.Blockdata {
			j, err := getItemBlockdataMap(m, block.Blockdata[i])
			if err != nil {
				return err
			}
			items[i] = j
		}
	default:
		return errors.Errorf("unknown data type found, %q", cmd.DataType)
	}

	if cmd.bd == nil {
		return cmd.printBlockdata(m, items)
	} else if err := cmd.saveBlockdata(m, items); err != nil {
		return err
	}

	return nil
}

func (cmd *BlockDownloadCommand) printBlockdata(m block.BlockdataMap, items []block.BlockdataMapItem) error {
	for i := range items {
		item := items[i]

		s := fmt.Sprintf("{\"height\": %d, \"data_type\": %q}\n", m.Height(), item.Type())
		if j, err := cmd.printBlockdataItem(item); err != nil {
			return err
		} else if b, err := cmd.jsonenc.Marshal(j); err != nil {
			return err
		} else {
			s += string(b)
		}
		_, _ = fmt.Fprintln(os.Stdout, s)
	}

	return nil
}

func (cmd *BlockDownloadCommand) printBlockdataItem(item block.BlockdataMapItem) (interface{}, error) {
	r, err := cmd.requestBlockdata(item)
	if err != nil {
		return nil, err
	}

	writer := blockdata.NewDefaultWriter(cmd.jsonenc)

	switch item.Type() {
	case block.BlockdataManifest:
		return writer.ReadManifest(r)
	case block.BlockdataOperations:
		return writer.ReadOperations(r)
	case block.BlockdataOperationsTree:
		return writer.ReadOperationsTree(r)
	case block.BlockdataStates:
		return writer.ReadStates(r)
	case block.BlockdataStatesTree:
		return writer.ReadStatesTree(r)
	case block.BlockdataINITVoteproof:
		return writer.ReadINITVoteproof(r)
	case block.BlockdataACCEPTVoteproof:
		return writer.ReadACCEPTVoteproof(r)
	case block.BlockdataSuffrageInfo:
		return writer.ReadSuffrageInfo(r)
	case block.BlockdataProposal:
		return writer.ReadProposal(r)
	default:
		return nil, errors.Errorf("unknown data type found, %q", item.Type())
	}
}

func (cmd *BlockDownloadCommand) saveBlockdata(m block.BlockdataMap, items []block.BlockdataMapItem) error {
	session, err := cmd.bd.NewSession(m.Height())
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Cancel()
	}()

	b := filepath.Join(cmd.Save, localfs.HeightDirectory(m.Height()))
	if i, err := os.Stat(b); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(b, localfs.DefaultDirectoryPermission); err != nil {
			return err
		}
	} else if !i.IsDir() {
		return errors.Errorf("block directory, %q already exists", b)
	}

	for i := range items {
		if err := cmd.saveBlockdataItem(m, items[i], session); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *BlockDownloadCommand) saveBlockdataItem(
	m block.BlockdataMap,
	item block.BlockdataMapItem,
	session blockdata.Session,
) error {
	r, err := cmd.requestBlockdata(item)
	if err != nil {
		return err
	}

	i, err := session.Import(item.Type(), r)
	if err != nil {
		return errors.Wrap(err, "failed to import block data")
	}
	b := filepath.Join(cmd.Save, localfs.HeightDirectory(m.Height()))
	f := filepath.Join(b, filepath.Base(i))
	if err := os.Rename(i, f); err != nil {
		return err
	}

	cmd.Log().Info().Int64("height", m.Height().Int64()).Str("file", f).Msg("saved")

	return nil
}

func (cmd *BlockDownloadCommand) requestBlockdata(item block.BlockdataMapItem) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	if block.IsLocalBlockdataItem(item.URL()) {
		return cmd.channel.Blockdata(ctx, item)
	}

	return network.FetchBlockdataFromRemote(ctx, item)
}

func getItemBlockdataMap(m block.BlockdataMap, dataType string) (block.BlockdataMapItem, error) {
	var item block.BlockdataMapItem
	switch dataType {
	case block.BlockdataManifest:
		item = m.Manifest()
	case block.BlockdataOperations:
		item = m.Operations()
	case block.BlockdataOperationsTree:
		item = m.OperationsTree()
	case block.BlockdataStates:
		item = m.States()
	case block.BlockdataStatesTree:
		item = m.StatesTree()
	case block.BlockdataINITVoteproof:
		item = m.INITVoteproof()
	case block.BlockdataACCEPTVoteproof:
		item = m.ACCEPTVoteproof()
	case block.BlockdataSuffrageInfo:
		item = m.SuffrageInfo()
	case block.BlockdataProposal:
		item = m.Proposal()
	default:
		return nil, errors.Errorf("unknown data type found, %q", dataType)
	}

	return item, nil
}

func requestBlockdataMaps(heights []base.Height, callback func([]base.Height) error) error {
	wk := util.NewErrgroupWorker(context.Background(), int64(quicnetwork.LimitRequestByHeights))
	defer wk.Close()

	limit := quicnetwork.LimitRequestByHeights
	l := len(heights) / limit
	if len(heights)%limit != 0 {
		l++
	}

	go func() {
		defer wk.Done()

		for i := 0; i < l; i++ {
			e := (i + 1) * limit
			if e > len(heights) {
				e = len(heights)
			}
			hs := heights[i*limit : e]

			if err := wk.NewJob(func(context.Context, uint64) error {
				return callback(hs)
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}

func requestBlockdata(maps []block.BlockdataMap, callback func(block.BlockdataMap) error) error {
	wk := util.NewErrgroupWorker(context.Background(), 10)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range maps {
			m := maps[i]

			if err := wk.NewJob(func(context.Context, uint64) error {
				return callback(m)
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}
