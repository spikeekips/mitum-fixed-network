package process

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameLocalNode = "local_node"

var ProcessorLocalNode pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameLocalNode,
		[]string{ProcessNameConfig},
		ProcessLocalNode,
	); err != nil {
		panic(err)
	} else {
		ProcessorLocalNode = i
	}
}

func ProcessLocalNode(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	local := network.NewLocalNode(conf.Address(), conf.Privatekey(), conf.Network().URL().String())

	nodepool := network.NewNodepool(local)
	log.Debug().Str("added_node", local.Address().String()).Msg("local node added to nodepool")

	ctx = context.WithValue(ctx, ContextValueNodepool, nodepool)

	return context.WithValue(ctx, ContextValueLocalNode, local), nil
}
