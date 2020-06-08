package cmds

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type NodeInfoCommand struct {
	URL *url.URL `arg:"" name:"node url" help:"remote mitum url" required:""`
}

func (cmd *NodeInfoCommand) Run(log logging.Logger) error {
	log.Debug().Interface("node_url", cmd.URL).Msg("trying to get node info")

	var channel network.NetworkChannel
	if ch, err := loadNodeChannel(cmd.URL, log); err != nil {
		return err
	} else {
		channel = ch
	}
	log.Debug().Msg("network channel loaded")

	log.Debug().Msg("trying to get node info")

	if n, err := channel.NodeInfo(); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(n))
	}

	return nil
}
