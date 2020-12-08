package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
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
	case config.FixedProposerSuffrage:
		var found bool
		if t.Proposer.Equal(local.Node().Address()) {
			found = true
		} else {
			local.Nodes().Traverse(func(node network.Node) bool {
				address := node.Address()
				if !found && t.Proposer.Equal(address) {
					found = true
				}

				return true
			})
		}

		if !found {
			return ctx, xerrors.Errorf("proposer not found in suffrage nodes or local")
		}

		sf = base.NewFixedSuffrage(t.Proposer, nil)
	case config.RoundrobinSuffrage:
		sf = NewRoundrobinSuffrage(local, t.CacheSize)
	}

	if err := sf.Initialize(); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValueSuffrage, sf), nil
}