package cmds

import (
	"fmt"
	"net/url"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoCommand struct {
	*BaseCommand
	URL *url.URL `arg:"" name:"node url" help:"remote mitum url" required:""`
}

func NewNodeInfoCommand() NodeInfoCommand {
	return NodeInfoCommand{
		BaseCommand: NewBaseCommand("node_info"),
	}
}

func (cmd *NodeInfoCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL).Msg("trying to get node info")

	encs := cmd.Encoders()
	if encs == nil {
		if i, err := cmd.LoadEncoders(nil); err != nil {
			return err
		} else {
			encs = i
		}
	}

	var channel network.Channel
	if ch, err := process.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}
	cmd.Log().Debug().Msg("network channel loaded")

	cmd.Log().Debug().Msg("trying to get node info")

	if n, err := channel.NodeInfo(); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(n))
	}

	return nil
}
