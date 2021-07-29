package process

import (
	"context"

	"github.com/spikeekips/mitum/base/node"
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

		no := node.NewRemote(conf.Address(), conf.Publickey())
		var ch network.Channel
		if ci := conf.ConnInfo(); ci != nil {
			i, err := LoadNodeChannel(ci, encs, policy.NetworkConnectionTimeout())
			if err != nil {
				return ctx, err
			}
			ch = i
		}

		if err := nodepool.Add(no, ch); err != nil {
			return ctx, err
		}
		log.Debug().Str("added_node", no.Address().String()).Msg("node added to nodepool")
	}

	return ctx, nil
}
