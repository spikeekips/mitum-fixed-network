package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

const ProcessNameSuffrage = "suffrage"

var ProcessorSuffrage pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameSuffrage,
		[]string{
			ProcessNameConfig,
			ProcessNameLocalNode,
		},
		ProcessSuffrage,
	); err != nil {
		panic(err)
	} else {
		ProcessorSuffrage = i
	}
}

func ProcessSuffrage(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var l config.LocalNode
	var conf config.Suffrage
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.Suffrage()
	}

	var sf base.Suffrage
	switch t := conf.(type) {
	case config.FixedSuffrage:
		if s, err := processFixedSuffrage(ctx, t); err != nil {
			return ctx, err
		} else {
			sf = s
		}
	case config.RoundrobinSuffrage:
		if s, err := processRoundrobinSuffrage(ctx, t); err != nil {
			return ctx, err
		} else {
			sf = s
		}
	}

	if err := sf.Initialize(); err != nil {
		return ctx, err
	}

	log.Debug().Interface("suffrage_nodes", sf.Nodes()).Msg("suffrage done")

	return context.WithValue(ctx, ContextValueSuffrage, sf), nil
}

func processFixedSuffrage(ctx context.Context, conf config.FixedSuffrage) (base.Suffrage, error) {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var nodes []base.Address
	if len(conf.Nodes) < 1 {
		nodes = nodepool.Addresses()
	} else {
		for i := range conf.Nodes {
			c := conf.Nodes[i]
			if !nodepool.Exists(c) {
				return nil, xerrors.Errorf("unknown node of fixed-suffrage found, %q", c)
			}
		}

		nodes = conf.Nodes
	}

	if conf.NumberOfActing < 1 {
		return nil, xerrors.Errorf("number-of-acting should be over zero")
	}

	return NewFixedSuffrage(conf.Proposer, nodes, conf.NumberOfActing, conf.CacheSize)
}

func processRoundrobinSuffrage(ctx context.Context, conf config.RoundrobinSuffrage) (base.Suffrage, error) {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var nodes []base.Address
	if len(conf.Nodes) < 1 {
		nodes = nodepool.Addresses()
	} else {
		for i := range conf.Nodes {
			c := conf.Nodes[i]
			if !nodepool.Exists(c) {
				return nil, xerrors.Errorf("unknown node of fixed-suffrage found, %q", c)
			}
		}

		nodes = conf.Nodes
	}

	var st storage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return nil, err
	}

	if conf.NumberOfActing < 1 {
		return nil, xerrors.Errorf("number-of-acting should be over zero")
	}

	return NewRoundrobinSuffrage(
		nodes,
		conf.NumberOfActing,
		conf.CacheSize,
		func(height base.Height) (valuehash.Hash, error) {
			switch m, found, err := st.ManifestByHeight(height); {
			case err != nil:
				return nil, err
			case !found:
				return nil, storage.NotFoundError.Errorf("manifest not found for suffrage")
			default:
				return m.Hash(), nil
			}
		},
	)
}
