package cmds

import (
	"fmt"
	"net/url"
	"os"

	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

var BlocksVars = kong.Vars{
	"blocks_request_type": "manifest",
}

// TODO download and store blocks

type BlocksCommand struct {
	*BaseCommand
	URL     *url.URL `arg:"" name:"node url" help:"remote mitum url" required:""`
	Heights []int64  `arg:"" name:"height" help:"block height of blocks" required:""`
	Type    string   `name:"type" help:"{block manifest}" default:"${blocks_request_type}"`
	heights []base.Height
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

	cmd.Log().Debug().Msg("trying to get blocks thru channel")
	if i, err := cmd.requestByHeights(); err != nil {
		return err
	} else { // TODO save to local fs
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(i))
	}

	return nil
}

func (cmd *BlocksCommand) prepare() error {
	switch t := cmd.Type; t {
	case "block", "manifest":
	default:
		return xerrors.Errorf("unknown --type, %q found", t)
	}

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

	return nil
}

func (cmd *BlocksCommand) requestByHeights() (interface{}, error) {
	encs := cmd.Encoders()
	if encs == nil {
		if i, err := cmd.LoadEncoders(nil); err != nil {
			return nil, err
		} else {
			encs = i
		}
	}

	var channel network.Channel
	if ch, err := process.LoadNodeChannel(cmd.URL, encs); err != nil {
		return nil, err
	} else {
		channel = ch
	}
	cmd.Log().Debug().Msg("network channel loaded")

	switch cmd.Type {
	case "block":
		return channel.Blocks(cmd.heights)
	case "manifest":
		return channel.Manifests(cmd.heights)
	default:
		return nil, xerrors.Errorf("unknown request: %s", cmd.Type)
	}
}
