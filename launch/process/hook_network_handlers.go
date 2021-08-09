package process

import (
	"context"
	"io"
	"sort"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

const HookNameSetNetworkHandlers = "set_network_handlers"

func HookSetNetworkHandlers(ctx context.Context) (context.Context, error) {
	if sn, err := SettingNetworkHandlersFromContext(ctx); err != nil {
		return ctx, err
	} else if err := sn.Set(); err != nil {
		return ctx, err
	} else {
		return ctx, nil
	}
}

type SettingNetworkHandlers struct {
	version   util.Version
	ctx       context.Context
	conf      config.LocalNode
	database  storage.Database
	blockData blockdata.BlockData
	policy    *isaac.LocalPolicy
	nodepool  *network.Nodepool
	suffrage  base.Suffrage
	states    states.States
	network   network.Server
	sealCache cache.Cache
	logger    *zerolog.Logger
}

func SettingNetworkHandlersFromContext(ctx context.Context) (*SettingNetworkHandlers, error) { // nolint:funlen
	var version util.Version
	if err := LoadVersionContextValue(ctx, &version); err != nil {
		return nil, err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return nil, err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var st storage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	var consensusStates states.States
	if err := LoadConsensusStatesContextValue(ctx, &consensusStates); err != nil {
		return nil, err
	}

	sealCache, err := cache.NewCacheFromURI(conf.Network().SealCache().String())
	if err != nil {
		return nil, err
	}

	l := zerolog.Nop()
	logger := &l
	var nt network.Server
	if err := LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	} else if l, ok := nt.(logging.HasLogger); ok {
		logger = l.Log()
	}

	return &SettingNetworkHandlers{
		ctx:       ctx,
		version:   version,
		conf:      conf,
		database:  st,
		blockData: blockData,
		policy:    policy,
		nodepool:  nodepool,
		suffrage:  suffrage,
		states:    consensusStates,
		network:   nt,
		sealCache: sealCache,
		logger:    logger,
	}, nil
}

func (sn *SettingNetworkHandlers) Set() error {
	sn.network.SetHasSealHandler(sn.networkHandlerHasSeal())
	sn.network.SetGetSealsHandler(sn.networkHandlerGetSeals())
	sn.network.SetNewSealHandler(sn.networkhandlerNewSeal())
	sn.network.SetNodeInfoHandler(sn.networkHandlerNodeInfo())
	sn.network.SetBlockDataMapsHandler(sn.networkHandlerBlockDataMaps())
	sn.network.SetBlockDataHandler(sn.networkHandlerBlockData())

	lc := sn.nodepool.LocalChannel().(*network.DummyChannel)
	lc.SetNewSealHandler(sn.networkhandlerNewSeal())
	lc.SetGetSealsHandler(sn.networkHandlerGetSeals())
	lc.SetNodeInfoHandler(sn.networkHandlerNodeInfo())
	lc.SetBlockDataMapsHandler(sn.networkHandlerBlockDataMaps())
	lc.SetBlockDataHandler(sn.networkHandlerBlockData())

	sn.logger.Debug().Msg("local channel handlers binded")

	return nil
}

func (sn *SettingNetworkHandlers) networkHandlerHasSeal() network.HasSealHandler {
	return func(h valuehash.Hash) (bool, error) {
		return sn.database.HasSeal(h)
	}
}

func (sn *SettingNetworkHandlers) networkHandlerGetSeals() network.GetSealsHandler {
	return func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var sls []seal.Seal

		if err := sn.database.SealsByHash(hs, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
			sls = append(sls, sl)

			return true, nil
		}, true); err != nil {
			return nil, err
		}

		return sls, nil
	}
}

func (sn *SettingNetworkHandlers) networkhandlerNewSeal() network.NewSealHandler {
	return func(sl seal.Seal) error {
		sealChecker := isaac.NewSealChecker(
			sl,
			sn.database,
			sn.policy,
			sn.sealCache,
		)
		if err := util.NewChecker("network-new-seal-checker", []util.CheckerFunc{
			sealChecker.IsKnown,
			sealChecker.IsValid,
			sealChecker.IsValidOperationSeal,
		}).Check(); err != nil {
			if errors.Is(err, util.IgnoreError) {
				return nil
			}

			sn.logger.Error().Err(err).Msg("seal checking failed")

			return err
		}

		if t, ok := sl.(ballot.Ballot); ok {
			checker := isaac.NewBallotChecker(
				t,
				sn.database,
				sn.policy,
				sn.suffrage,
				sn.nodepool,
				sn.states.LastVoteproof(),
			)
			if err := util.NewChecker("network-new-ballot-checker", []util.CheckerFunc{
				checker.IsFromLocal,
				checker.InTimespan,
				checker.InSuffrage,
				checker.CheckSigning,
				checker.IsFromAliveNode,
				checker.CheckWithLastVoteproof,
				checker.CheckProposalInACCEPTBallot,
				checker.CheckVoteproof,
			}).Check(); err != nil {
				return err
			}
		}

		go func() {
			_ = sn.states.NewSeal(sl)
		}()

		return nil
	}
}

func (sn *SettingNetworkHandlers) networkHandlerNodeInfo() network.NodeInfoHandler {
	return func() (network.NodeInfo, error) {
		var manifest block.Manifest
		if m, found, err := sn.database.LastManifest(); err != nil {
			return nil, err
		} else if found {
			manifest = m
		}

		suffrageNodes := sn.suffrage.Nodes()
		nodes := make([]network.RemoteNode, len(suffrageNodes))
		for i := range suffrageNodes {
			n, ch, found := sn.nodepool.Node(suffrageNodes[i])
			if !found {
				return nil, errors.Errorf("suffrage node, %q not found", n.Address())
			}

			var connInfo network.ConnInfo
			if ch != nil {
				connInfo = ch.ConnInfo()
			}

			nodes[i] = network.NewRemoteNode(n, connInfo)
		}

		return network.NewNodeInfoV0(
			sn.nodepool.LocalNode(),
			sn.policy.NetworkID(),
			sn.states.State(),
			manifest,
			sn.version,
			sn.conf.Network().ConnInfo().URL().String(),
			sn.policy.Config(),
			nodes,
			sn.suffrage,
		), nil
	}
}

func (sn *SettingNetworkHandlers) networkHandlerBlockDataMaps() network.BlockDataMapsHandler {
	return func(heights []base.Height) ([]block.BlockDataMap, error) {
		sort.Slice(heights, func(i, j int) bool {
			return heights[i] < heights[j]
		})

		var filtered []base.Height
		founds := map[base.Height]struct{}{}
		for i := range heights {
			h := heights[i]
			if _, f := founds[h]; f {
				continue
			}

			founds[h] = struct{}{}
			filtered = append(filtered, h)
		}

		maps := make([]block.BlockDataMap, len(filtered))
		for i := range filtered {
			switch m, found, err := sn.database.BlockDataMap(filtered[i]); {
			case !found:
				continue
			case err != nil:
				return nil, err
			default:
				maps[i] = m
			}
		}

		return maps, nil
	}
}

func (sn *SettingNetworkHandlers) networkHandlerBlockData() network.BlockDataHandler {
	return func(p string) (io.Reader, func() error, error) {
		i, err := sn.blockData.FS().Open(p)
		if err != nil {
			return nil, func() error { return nil }, err
		}
		return i, i.Close, nil
	}
}
