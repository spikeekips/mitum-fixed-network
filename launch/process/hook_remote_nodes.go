package process

import (
	"context"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util/encoder"
)

const HookNameRemoteNodes = "remote_nodes"

// HookRemoteNodes generates the node list of local node. It does not include
// the local node itself.
func HookRemoteNodes(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var nodeConfigs []config.RemoteNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		nodeConfigs = l.Nodes()
	}

	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return ctx, err
	}

	for i := range nodeConfigs {
		conf := nodeConfigs[i]

		node := isaac.NewRemoteNode(conf.Address(), conf.Publickey(), conf.URL().String())
		if ch, err := LoadNodeChannel(conf.URL(), encs); err != nil {
			return ctx, err
		} else {
			_ = node.SetChannel(ch)
		}

		if err := local.Nodes().Add(node); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
