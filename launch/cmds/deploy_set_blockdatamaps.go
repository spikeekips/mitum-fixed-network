package cmds

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
)

type SetBlockdataMapsCommand struct {
	*BaseCommand
	DeployKey string   `arg:"" name:"deploy key"`
	File      *os.File `arg:"" name:"maps file" help:"set blockdatamap file"`
	*NodeConnectFlags
	client *quicnetwork.QuicClient
}

func NewSetBlockdataMapsCommand() SetBlockdataMapsCommand {
	return SetBlockdataMapsCommand{
		BaseCommand:      NewBaseCommand("deploy-set-blockdatamaps"),
		NodeConnectFlags: &NodeConnectFlags{},
	}
}

func (cmd *SetBlockdataMapsCommand) Run(version util.Version) error {
	cmd.BaseCommand.LogOutput = os.Stderr

	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
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
	var maps []block.BlockdataMap
	if i, err := cmd.loadMaps(); err != nil {
		return errors.Wrap(err, "failed to load maps from file")
	} else if n := len(maps); n > deploy.LimitBlockdataMaps {
		return errors.Errorf("too many maps over %d > %d", n, deploy.LimitBlockdataMaps)
	} else {
		maps = i

		cmd.Log().Debug().Int("maps", len(i)).Msg("maps loaded")

		for i := range maps {
			heights = append(heights, maps[i].Height())
		}
	}

	if err := cmd.request(maps); err != nil {
		var pr network.Problem
		if errors.As(err, &pr) {
			cmd.Log().Error().Interface("problem", pr).Msg("failed")
		}

		return err
	}

	cmd.Log().Info().Interface("heights", heights).Msg("blockdatamaps updated")

	return nil
}

func (cmd *SetBlockdataMapsCommand) loadMaps() ([]block.BlockdataMap, error) {
	if _, err := cmd.File.Seek(0, 0); err != nil {
		return nil, err
	}

	founds := map[base.Height]bool{}
	var maps []block.BlockdataMap
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

func (cmd *SetBlockdataMapsCommand) loadMap(b []byte) (block.BlockdataMap, error) {
	var m block.BaseBlockdataMap
	if i, err := cmd.JSONEncoder().Decode(b); err != nil {
		return nil, err
	} else if j, ok := i.(block.BaseBlockdataMap); !ok {
		return nil, errors.Errorf("expected block.BlockdataMap, not %T", i)
	} else {
		m = j
	}

	um := block.NewBaseBlockdataMap(m.Writer(), m.Height()).SetBlock(m.Block())
	items := m.Items()
	for i := range items {
		j, err := um.SetItem(items[i])
		if err != nil {
			return nil, err
		}
		um = j
	}

	var nm block.BlockdataMap
	if i, err := um.UpdateHash(); err != nil {
		return nil, errors.Wrap(err, "failed to update hash")
	} else if err := i.IsValid(nil); err != nil {
		return nil, errors.Wrap(err, "failed to update hash")
	} else {
		nm = i
	}

	i, err := cmd.JSONEncoder().Marshal(nm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal map")
	}
	_, _ = fmt.Fprintln(os.Stdout, string(i))

	return nm, nil
}

func (cmd *SetBlockdataMapsCommand) request(maps []block.BlockdataMap) error {
	body, err := cmd.JSONEncoder().Marshal(maps)
	if err != nil {
		return err
	}

	u := *cmd.URL
	u.Path = filepath.Join(u.Path, deploy.QuicHandlerPathSetBlockdataMaps)

	headers := http.Header{}
	headers.Set("Authorization", filepath.Join(u.Path, deploy.QuicHandlerPathSetBlockdataMaps))

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	res, c, err := cmd.client.Request(ctx, cmd.Timeout, u.String(), "POST", body, headers)
	if err != nil {
		return errors.Wrap(err, "failed to request")
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

		return errors.Errorf("failed to set blockdatamaps")
	}

	return nil
}
