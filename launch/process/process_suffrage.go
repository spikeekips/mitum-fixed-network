package process

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
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
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Suffrage()

	var sf base.Suffrage
	switch t := conf.(type) {
	case config.FixedSuffrage:
		s, err := processFixedSuffrage(ctx, t)
		if err != nil {
			return ctx, err
		}
		sf = s
	case config.RoundrobinSuffrage:
		s, err := processRoundrobinSuffrage(ctx, t)
		if err != nil {
			return ctx, err
		}
		sf = s
	case config.EmptySuffrage:
		sf = EmptySuffrage{}
	}

	if err := sf.Initialize(); err != nil {
		return ctx, err
	}

	if err := cleanSuffrageChannels(ctx, sf); err != nil {
		return ctx, err
	}

	log.Log().Debug().
		Bool("in_suffrage", sf.IsInside(l.Address())).
		Interface("suffrage_nodes", sf.Nodes()).
		Msg("suffrage done")

	return context.WithValue(ctx, ContextValueSuffrage, sf), nil
}

// cleanSuffrageChannels cleans channel of suffrage node if local is suffrage
// node.
func cleanSuffrageChannels(ctx context.Context, sf base.Suffrage) error {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return err
	}

	if !sf.IsInside(nodepool.LocalNode().Address()) {
		return nil
	}

	var nodepoolerr error
	nodepool.TraverseRemotes(func(no base.Node, ch network.Channel) bool {
		addr := no.Address()
		if !sf.IsInside(addr) {
			return true
		}

		if err := nodepool.SetChannel(addr, nil); err != nil {
			nodepoolerr = err

			return false
		}

		return true
	})

	return nodepoolerr
}

func processFixedSuffrage(ctx context.Context, conf config.FixedSuffrage) (base.Suffrage, error) {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	if len(conf.Nodes()) < 1 {
		return nil, errors.Errorf("empty nodes for suffrage")
	}

	for i := range conf.Nodes() {
		c := conf.Nodes()[i]
		if !nodepool.Exists(c) {
			return nil, errors.Errorf("unknown node of fixed-suffrage found, %q", c)
		}
	}

	return NewFixedSuffrage(conf.Proposer, conf.Nodes(), conf.NumberOfActing(), conf.CacheSize)
}

func processRoundrobinSuffrage(ctx context.Context, conf config.RoundrobinSuffrage) (base.Suffrage, error) {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	if len(conf.Nodes()) < 1 {
		return nil, errors.Errorf("empty nodes for suffrage")
	}
	for i := range conf.Nodes() {
		c := conf.Nodes()[i]
		if !nodepool.Exists(c) {
			return nil, errors.Errorf("unknown node of fixed-suffrage found, %q", c)
		}
	}

	var db storage.Database
	if err := LoadDatabaseContextValue(ctx, &db); err != nil {
		return nil, err
	}

	return NewRoundrobinSuffrage(
		conf.Nodes(),
		conf.NumberOfActing(),
		conf.CacheSize,
		func(height base.Height) (valuehash.Hash, error) {
			switch m, found, err := db.ManifestByHeight(height); {
			case err != nil:
				return nil, err
			case !found:
				return nil, util.NotFoundError.Errorf("manifest not found for suffrage")
			default:
				return m.Hash(), nil
			}
		},
	)
}
