package process

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameDiscovery = "discovery"

var ProcessorDiscovery pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameDiscovery,
		[]string{
			ProcessNameNetwork,
		},
		ProcessDiscovery,
	); err != nil {
		panic(err)
	} else {
		ProcessorDiscovery = i
	}
}

func ProcessDiscovery(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var local *node.Local
	if err := LoadLocalNodeContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	if !suffrage.IsInside(local.Address()) {
		log.Log().Debug().Msg("local is not suffrage node; discovery disabled")

		return ctx, nil
	}

	cis, err := processDiscoveryURLs(ctx)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, ContextValueDiscoveryConnInfos, cis)

	dis, err := processDiscovery(ctx)
	if err != nil {
		return ctx, err
	}

	if err := processDiscoveryDelegate(ctx, dis); err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, ContextValueDiscovery, dis)

	return ctx, nil
}

func processDiscoveryURLs(ctx context.Context) ([]network.ConnInfo, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var ln config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &ln); err != nil {
		return nil, err
	}
	connInfo := ln.Network().ConnInfo()

	var urls []*url.URL
	if err := config.LoadDiscoveryURLsContextValue(ctx, &urls); err != nil {
		if errors.Is(err, util.ContextValueNotFoundError) {
			return nil, nil
		}

		return nil, err
	}

	if len(urls) < 1 {
		log.Log().Debug().Msg("empty discovery urls")
	} else {
		log.Log().Debug().Interface("urls", urls).Msg("discovery urls")
	}

	var cis []network.ConnInfo // nolint:prealloc
	for i := range urls {
		u := urls[i]
		ci, err := parseCombinedNodeURL(u)
		if err != nil {
			return nil, errors.Wrap(err, "invalid discovery url")
		}

		if connInfo.URL().String() == ci.URL().String() {
			log.Log().Warn().Msg("local discovery url ignored")

			continue
		}

		cis = append(cis, ci)
	}

	return cis, nil
}

func processDiscovery(ctx context.Context) (discovery.Discovery, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var local *node.Local
	if err := LoadLocalNodeContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var ln config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &ln); err != nil {
		return nil, err
	}
	connInfo := ln.Network().ConnInfo()

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var networkID base.NetworkID
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return nil, err
	}
	networkID = l.NetworkID()

	var nt *quicnetwork.Server
	if err := util.LoadFromContextValue(ctx, ContextValueNetwork, &nt); err != nil {
		return nil, err
	}

	var nlog *logging.Logging
	if err := config.LoadNetworkLogContextValue(ctx, &nlog); err != nil {
		return nil, err
	}

	dis := memberlist.NewDiscovery(local, connInfo, networkID, nt.Encoder())
	_ = dis.SetLogging(nlog)

	if err := dis.Initialize(); err != nil {
		return nil, err
	}

	_ = nt.SetHandler(
		memberlist.DefaultDiscoveryPath,
		dis.Handler(memberlist.SuffrageHandlerFilter(suffrage, nodepool)),
	)

	return dis, nil
}

func processDiscoveryDelegate(ctx context.Context, dis discovery.Discovery) error {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return err
	}

	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return err
	}

	var nt *quicnetwork.Server
	if err := util.LoadFromContextValue(ctx, ContextValueNetwork, &nt); err != nil {
		return err
	}

	dg := discovery.NewNodepoolDelegate(nodepool, nt.Encoders(), policy.NetworkConnectionTimeout())
	_ = dg.SetLogging(log)

	_ = dis.SetNotifyJoin(dg.NotifyJoin).
		SetNotifyLeave(dg.NotifyLeave).
		SetNotifyUpdate(dg.NotifyUpdate)

	return nil
}

func parseCombinedNodeURL(u *url.URL) (network.HTTPConnInfo, error) {
	i, insecure, err := network.ParseCombinedNodeURL(u)
	if err != nil {
		return network.HTTPConnInfo{}, err
	}

	ci := network.NewHTTPConnInfo(i, insecure)
	return ci, ci.IsValid(nil)
}
