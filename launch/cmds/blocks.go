package cmds

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BlocksCommand struct {
	*BaseCommand
	URL        *url.URL      `arg:"" name:"node url" help:"remote mitum url" required:""`
	Heights    []int64       `arg:"" name:"height" help:"block height of blocks" required:""`
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	channel    network.Channel
	heights    []base.Height
}

func NewBlocksCommand() BlocksCommand {
	return BlocksCommand{
		BaseCommand: NewBaseCommand("blocks"),
	}
}

func (cmd *BlocksCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL.String()).Ints64("heights", cmd.Heights).Msg("trying to get blocks")

	if err := cmd.prepare(); err != nil {
		return err
	}

	cmd.Log().Debug().Msg("trying to get block data maps thru channel")

	limit := 20
	var heights []base.Height
	for i := range cmd.heights {
		if len(heights) != limit {
			heights = append(heights, cmd.heights[i])

			continue
		}

		if err := cmd.request(heights); err != nil {
			return err
		}

		heights = nil
	}

	if len(heights) > 0 {
		if err := cmd.request(heights); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *BlocksCommand) prepare() error {
	encs := cmd.Encoders()
	if encs == nil {
		if i, err := cmd.LoadEncoders(nil); err != nil {
			return err
		} else {
			encs = i
		}
	}

	if ch, err := process.LoadNodeChannel(cmd.URL, encs, cmd.Timeout, cmd.TLSInscure); err != nil {
		return err
	} else {
		cmd.channel = ch
	}

	cmd.Log().Debug().Msg("network channel loaded")

	var heights []base.Height // nolint
	for _, i := range cmd.Heights {
		h := base.Height(i)
		if err := h.IsValid(nil); err != nil {
			return err
		}

		var found bool
		for _, m := range heights {
			if h == m {
				found = true

				break
			}
		}
		if found {
			continue
		}

		heights = append(heights, h)
	}

	if len(heights) < 1 {
		return xerrors.Errorf("missing height")
	}

	cmd.heights = heights

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	return nil
}

func (cmd *BlocksCommand) request(heights []base.Height) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	if maps, err := cmd.channel.BlockDataMaps(ctx, heights); err != nil {
		return err
	} else {
		for i := range maps {
			_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(maps[i]))
		}
	}

	return nil
}
