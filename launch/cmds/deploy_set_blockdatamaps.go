package cmds

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type SetBlockDataMapsCommand struct {
	*BaseCommand
	DeployKey string   `arg:"" name:"deploy key"`
	File      *os.File `arg:"" name:"maps file" help:"set blockdatamap file"`
	*NodeConnectFlags
	client *quicnetwork.QuicClient
}

func NewSetBlockDataMapsCommand() SetBlockDataMapsCommand {
	return SetBlockDataMapsCommand{
		BaseCommand:      NewBaseCommand("deploy-set-blockdatamaps"),
		NodeConnectFlags: &NodeConnectFlags{},
	}
}

func (cmd *SetBlockDataMapsCommand) Run(version util.Version) error {
	cmd.BaseCommand.LogOutput = os.Stderr

	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	} else if _, err := cmd.LoadEncoders(nil, nil); err != nil {
		return err
	}

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL).Msg("deploy set-blockdatamaps")

	quicConfig := &quic.Config{HandshakeIdleTimeout: cmd.Timeout}
	i, err := quicnetwork.NewQuicClient(cmd.TLSInscure, quicConfig)
	if err != nil {
		return err
	}
	cmd.client = i

	var heights []base.Height
	var maps []block.BlockDataMap
	if i, err := cmd.loadMaps(); err != nil {
		return xerrors.Errorf("failed to load maps from file: %w", err)
	} else if n := len(maps); n > deploy.LimitBlockDataMaps {
		return xerrors.Errorf("too many maps over %d > %d", n, deploy.LimitBlockDataMaps)
	} else {
		maps = i

		cmd.Log().Debug().Int("maps", len(i)).Msg("maps loaded")

		for i := range maps {
			heights = append(heights, maps[i].Height())
		}
	}

	if err := cmd.request(maps); err != nil {
		var pr network.Problem
		if xerrors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	cmd.Log().Info().Interface("heights", heights).Msg("blockdatamaps updated")

	return nil
}

func (cmd *SetBlockDataMapsCommand) loadMaps() ([]block.BlockDataMap, error) {
	if _, err := cmd.File.Seek(0, 0); err != nil {
		return nil, err
	}

	founds := map[base.Height]bool{}
	var maps []block.BlockDataMap
	if err := util.Readlines(cmd.File, func(b []byte) error {
		if len(b) < 1 {
			return nil
		}

		if i, err := cmd.loadMap(b); err != nil {
			return err
		} else if _, found := founds[i.Height()]; found {
			cmd.Log().Debug().Interface("map", i).Msg("duplicated map found")

			return nil
		} else {
			founds[i.Height()] = true
			maps = append(maps, i)

			return nil
		}
	}); err != nil {
		return nil, err
	}

	return maps, nil
}

func (cmd *SetBlockDataMapsCommand) loadMap(b []byte) (block.BlockDataMap, error) {
	var m block.BaseBlockDataMap
	if i, err := cmd.JSONEncoder().Decode(b); err != nil {
		return nil, err
	} else if j, ok := i.(block.BaseBlockDataMap); !ok {
		return nil, xerrors.Errorf("expected block.BlockDataMap, not %T", i)
	} else {
		m = j
	}

	um := block.NewBaseBlockDataMap(m.Writer(), m.Height()).SetBlock(m.Block())
	items := m.Items()
	for i := range items {
		j, err := um.SetItem(items[i])
		if err != nil {
			return nil, err
		}
		um = j
	}

	var nm block.BlockDataMap
	if i, err := um.UpdateHash(); err != nil {
		return nil, xerrors.Errorf("failed to update hash: %w", err)
	} else if err := i.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("failed to update hash: %w", err)
	} else {
		nm = i
	}

	i, err := cmd.JSONEncoder().Marshal(nm)
	if err != nil {
		return nil, xerrors.Errorf("failed to marshal map: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(i))

	return nm, nil
}

func (cmd *SetBlockDataMapsCommand) request(maps []block.BlockDataMap) error {
	body, err := cmd.JSONEncoder().Marshal(maps)
	if err != nil {
		return err
	}

	u := *cmd.URL
	u.Path = filepath.Join(u.Path, deploy.QuicHandlerPathSetBlockDataMaps)

	headers := http.Header{}
	headers.Set("Authorization", filepath.Join(u.Path, deploy.QuicHandlerPathSetBlockDataMaps))

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	res, c, err := cmd.client.Request(ctx, cmd.Timeout, u.String(), "POST", body, headers)
	if err != nil {
		return xerrors.Errorf("failed to request: %w", err)
	}
	defer func() {
		_ = c()
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if i, err := network.LoadProblemFromResponse(res); err == nil {
			cmd.Log().Debug().Interface("response", res).Interface("problem", i).Msg("failed to request")

			return i
		}

		cmd.Log().Debug().Interface("response", res).Msg("failed to set blockdatamaps")

		return xerrors.Errorf("failed to set blockdatamaps")
	}

	return nil
}
