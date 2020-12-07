package process

import (
	"context"
	"sort"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
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
	version         util.Version
	ctx             context.Context
	conf            config.LocalNode
	local           *isaac.Local
	storage         storage.Storage
	blockfs         *storage.BlockFS
	suffrage        base.Suffrage
	consensusStates *isaac.ConsensusStates
	network         network.Server
	sealCache       cache.Cache
	logger          logging.Logger
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

	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var st storage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var blockfs *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockfs); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	var consensusStates *isaac.ConsensusStates
	if err := LoadConsensusStatesContextValue(ctx, &consensusStates); err != nil {
		return nil, err
	}

	var sealCache cache.Cache
	// TODO sealCache also can be configurable
	if ca, err := cache.NewGCache("lru", 100*100, time.Minute*3); err != nil {
		return nil, err
	} else {
		sealCache = ca
	}

	var nt network.Server
	var logger logging.Logger = logging.NilLogger
	if err := LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	} else if l, ok := nt.(logging.HasLogger); ok {
		logger = l.Log()
	}

	return &SettingNetworkHandlers{
		ctx:             ctx,
		version:         version,
		conf:            conf,
		local:           local,
		storage:         st,
		blockfs:         blockfs,
		suffrage:        suffrage,
		consensusStates: consensusStates,
		network:         nt,
		sealCache:       sealCache,
		logger:          logger,
	}, nil
}

func (sn *SettingNetworkHandlers) Set() error {
	sn.network.SetHasSealHandler(sn.networkHandlerHasSeal())
	sn.network.SetGetSealsHandler(sn.networkHandlerGetSeals())
	sn.network.SetNewSealHandler(sn.networkhandlerNewSeal())
	sn.network.SetGetManifestsHandler(sn.networkhandlerGetManifests())
	sn.network.SetGetBlocksHandler(sn.networkhandlerGetBlocks())
	sn.network.SetNodeInfoHandler(sn.networkHandlerNodeInfo())

	return nil
}

func (sn *SettingNetworkHandlers) networkHandlerHasSeal() network.HasSealHandler {
	return func(h valuehash.Hash) (bool, error) {
		return sn.storage.HasSeal(h)
	}
}

func (sn *SettingNetworkHandlers) networkHandlerGetSeals() network.GetSealsHandler {
	return func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var sls []seal.Seal

		if err := sn.storage.SealsByHash(hs, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
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
		sealChecker := isaac.NewSealValidationChecker(
			sl,
			sn.storage,
			sn.local.Policy(),
			sn.sealCache,
		)
		if err := util.NewChecker("network-new-seal-checker", []util.CheckerFunc{
			sealChecker.CheckIsKnown,
			sealChecker.CheckIsValid,
			func() (bool, error) {
				// NOTE stores seal regardless further checkings.
				if err := sn.storage.NewSeals([]seal.Seal{sl}); err != nil {
					if !xerrors.Is(err, storage.DuplicatedError) {
						return false, err
					}
				}

				return true, nil
			},
		}).Check(); err != nil {
			if xerrors.Is(err, util.CheckerNilError) {
				sn.logger.Debug().Msg(err.Error())

				return nil
			}

			return err
		}

		if t, ok := sl.(ballot.Ballot); ok {
			if checker, err := isaac.NewBallotChecker(t, sn.local, sn.suffrage); err != nil {
				return err
			} else if err := util.NewChecker("network-new-ballot-checker", []util.CheckerFunc{
				checker.CheckIsInSuffrage,
				checker.CheckSigning,
				checker.CheckWithLastBlock,
				checker.CheckProposal,
				checker.CheckVoteproof,
			}).Check(); err != nil {
				return err
			}
		}

		sn.consensusStates.NewSeal(sl)

		return nil
	}
}

func (sn *SettingNetworkHandlers) networkhandlerGetManifests() network.GetManifestsHandler {
	return func(heights []base.Height) ([]block.Manifest, error) {
		sort.Slice(heights, func(i, j int) bool {
			return heights[i] < heights[j]
		})

		var manifests []block.Manifest
		fetched := map[base.Height]struct{}{}
		for _, h := range heights {
			if _, found := fetched[h]; found {
				continue
			}

			fetched[h] = struct{}{}

			switch m, found, err := sn.storage.ManifestByHeight(h); {
			case !found:
				continue
			case err != nil:
				return nil, err
			default:
				manifests = append(manifests, m)
			}
		}

		return manifests, nil
	}
}

func (sn *SettingNetworkHandlers) networkhandlerGetBlocks() network.GetBlocksHandler {
	return func(heights []base.Height) ([]block.Block, error) {
		sort.Slice(heights, func(i, j int) bool {
			return heights[i] < heights[j]
		})

		var blocks []block.Block
		for _, h := range heights {
			if blk, err := sn.blockfs.Load(h); err != nil {
				if xerrors.Is(err, storage.NotFoundError) {
					break
				}

				return nil, err
			} else {
				blocks = append(blocks, blk)
			}
		}

		return blocks, nil
	}
}

func (sn *SettingNetworkHandlers) networkHandlerNodeInfo() network.NodeInfoHandler {
	return func() (network.NodeInfo, error) {
		var state base.State = base.StateUnknown
		if handler := sn.consensusStates.ActiveHandler(); handler != nil {
			state = handler.State()
		}

		var manifest block.Manifest
		if m, found, err := sn.storage.LastManifest(); err != nil {
			return nil, err
		} else if found {
			manifest = m
		}

		suffrage := make([]base.Node, sn.local.Nodes().Len())
		var i int
		sn.local.Nodes().Traverse(func(n network.Node) bool {
			suffrage[i] = n
			i++

			return true
		})

		return network.NewNodeInfoV0(
			sn.local.Node(),
			sn.local.Policy().NetworkID(),
			state,
			manifest,
			sn.version,
			sn.conf.Network().URL().String(),
			sn.local.Policy().Policy(),
			sn.local.Policy().Config(),
			suffrage,
		), nil
	}
}
