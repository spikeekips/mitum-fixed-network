package process

import (
	"context"

	"github.com/spikeekips/mitum/base/node"
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
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	no := node.NewLocal(conf.Address(), conf.Privatekey())
	ch := network.NewDummyChannel(network.NewNilConnInfo("local://"))

	nodepool := network.NewNodepool(no, ch)
	log.Log().Debug().Stringer("added_node", no.Address()).Msg("local node added to nodepool")

	ctx = context.WithValue(ctx, ContextValueNodepool, nodepool)

	return context.WithValue(ctx, ContextValueLocalNode, no), nil
}
