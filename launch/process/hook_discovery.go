package process

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

const HookNameStartDiscovery = "start-discovery"

func HookStartDiscovery(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var local *node.Local
	if err := LoadLocalNodeContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	if !suffrage.IsInside(local.Address()) {
		return ctx, nil
	}

	var dis *memberlist.Discovery
	if err := util.LoadFromContextValue(ctx, ContextValueDiscovery, &dis); err != nil {
		return ctx, err
	}

	if err := dis.Start(); err != nil {
		return ctx, err
	}

	var cis []memberlist.ConnInfo
	if err := LoadDiscoveryConnInfosContextValue(ctx, &cis); err != nil {
		if !xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if len(cis) < 1 {
		log.Debug().Msg("empty discovery urls")

		return ctx, nil
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return ctx, err
	}

	// NOTE join network
	if err := joinDiscovery(nodepool, suffrage, dis, cis, 2, log); err != nil {
		if !xerrors.Is(err, memberlist.JoiningCanceledError) {
			return ctx, err
		}

		log.Error().Err(err).Msg("failed to join network; keep trying")

		go keepDiscoveryJoining(nodepool, suffrage, dis, cis, log)
	}

	return ctx, nil
}

func joinDiscovery(
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	dis *memberlist.Discovery,
	cis []memberlist.ConnInfo,
	maxretry int,
	log logging.Logger,
) error {
	// NOTE join network
	if err := dis.Join(cis, maxretry); err != nil {
		return err
	}

	joined := dis.Nodes()
	if len(joined) < 1 {
		return memberlist.JoiningCanceledError.Errorf("failed to join network; empty joined nodes")
	}

	var alives []map[string]interface{}
	nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		if !suffrage.IsInside(no.Address()) {
			return true
		}

		alives = append(alives, map[string]interface{}{
			no.Address().String(): ch.ConnInfo(),
		})

		return true
	})

	if len(alives) < 1 {
		return memberlist.JoiningCanceledError.Errorf("any nodes did not join, except local")
	}

	log.Debug().Interface("joined", alives).Msg("joined network")

	return nil
}

func keepDiscoveryJoining(
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	dis *memberlist.Discovery,
	cis []memberlist.ConnInfo,
	log logging.Logger,
) {
	for {
		err := joinDiscovery(nodepool, suffrage, dis, cis, -1, log)
		if err == nil {
			break
		}

		log.Error().Err(err).Msg("failed to join network; keep retrying")

		<-time.After(time.Second * 2)
	}
}
