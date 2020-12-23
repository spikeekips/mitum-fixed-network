package process

import (
	"context"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
)

const ProcessNameLocal = "local"

var ProcessorLocal pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameLocal,
		[]string{ProcessNameConfig, ProcessNameStorage, ProcessNameBlockFS},
		ProcessLocal,
	); err != nil {
		panic(err)
	} else {
		ProcessorLocal = i
	}
}

func ProcessLocal(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	node := network.NewLocalNode(conf.Address(), conf.Privatekey(), conf.Network().URL().String())

	var blockfs *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockfs); err != nil {
		return ctx, err
	}

	var st storage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	if ls, err := isaac.NewLocal(st, blockfs, node, conf.NetworkID()); err != nil {
		return ctx, err
	} else if err := ls.Initialize(); err != nil {
		return ctx, err
	} else {
		ctx = context.WithValue(ctx, ContextValueLocal, ls)

		return ctx, nil
	}
}
