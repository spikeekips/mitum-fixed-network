package process

import (
	"context"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

const HookNameNodepool = "nodepool"

// HookNodepool generates the node list of local node. It does not include the
// local node itself.
func HookNodepool(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}

	nodeConfigs := l.Nodes()

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return ctx, err
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return ctx, err
	}

	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}

	for i := range nodeConfigs {
		conf := nodeConfigs[i]

		node := network.NewRemoteNode(conf.Address(), conf.Publickey(), conf.URL().String())
		ch, err := LoadNodeChannel(
			conf.URL(),
			encs,
			policy.NetworkConnectionTimeout(),
		)
		if err != nil {
			return ctx, err
		}
		_ = node.SetChannel(ch)

		if err := nodepool.Add(node); err != nil {
			return ctx, err
		}
		log.Debug().Str("added_node", node.Address().String()).Msg("node added to nodepool")
	}

	return ctx, nil
}
