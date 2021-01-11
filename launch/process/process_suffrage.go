package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

const ProcessNameSuffrage = "suffrage"

var ProcessorSuffrage pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameSuffrage,
		[]string{
			ProcessNameConfig,
		},
		ProcessSuffrage,
	); err != nil {
		panic(err)
	} else {
		ProcessorSuffrage = i
	}
}

func ProcessSuffrage(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var conf config.Suffrage
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.Suffrage()
	}

	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return ctx, err
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

	return context.WithValue(ctx, ContextValueSuffrage, sf), nil
}

func processFixedSuffrage(ctx context.Context, conf config.FixedSuffrage) (base.Suffrage, error) {
	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return nil, err
	}

	// NOTE check proposer
	if conf.Proposer != nil {
		var found bool
		if conf.Proposer.Equal(local.Node().Address()) {
			found = true
		} else {
			local.Nodes().Traverse(func(node network.Node) bool {
				address := node.Address()
				if !found && conf.Proposer.Equal(address) {
					found = true
				}

				return true
			})
		}

		if !found {
			return nil, xerrors.Errorf("proposer not found in suffrage nodes or local")
		}
	}

	// NOTE check nodes
	for i := range conf.Nodes {
		c := conf.Nodes[i]

		if !local.Node().Address().Equal(c) {
			var found bool
			local.Nodes().Traverse(func(n network.Node) bool {
				if n.Address().Equal(c) {
					found = true

					return false
				}

				return true
			})

			if !found {
				return nil, xerrors.Errorf("unknown node of fixed-suffrage found, %q", c)
			}
		}
	}

	return NewFixedSuffrage(local, conf.CacheSize, conf.Proposer, conf.Nodes)
}

func processRoundrobinSuffrage(ctx context.Context, conf config.RoundrobinSuffrage) (base.Suffrage, error) {
	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var st storage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return nil, err
	}

	if conf.NumberOfActing < 1 {
		return nil, xerrors.Errorf("number-of-acting should be over zero")
	}

	return NewRoundrobinSuffrage(
		local,
		conf.CacheSize,
		conf.NumberOfActing,
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
	), nil
}
