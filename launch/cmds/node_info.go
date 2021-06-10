package cmds

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoCommand struct {
	*BaseCommand
	URL        *url.URL      `arg:"" name:"node url" help:"remote mitum url" required:"true"`
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
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

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	cmd.Log().Debug().Interface("node_url", cmd.URL).Msg("trying to get node info")

	encs := cmd.Encoders()
	if encs == nil {
		i, err := cmd.LoadEncoders(nil, nil)
		if err != nil {
			return err
		}
		encs = i
	}

	if cmd.TLSInscure {
		query := cmd.URL.Query()
		query.Set("insecure", "true")
		cmd.URL.RawQuery = query.Encode()
	}

	channel, err := process.LoadNodeChannel(cmd.URL, encs, cmd.Timeout)
	if err != nil {
		return err
	}
	cmd.Log().Debug().Msg("network channel loaded")

	cmd.Log().Debug().Msg("trying to get node info")

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	n, err := channel.NodeInfo(ctx)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(n))

	return nil
}
