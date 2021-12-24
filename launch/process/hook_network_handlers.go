package process

import (
	"context"
	"io"
	"sort"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
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
	conf      config.LocalNode
	database  storage.Database
	blockdata blockdata.Blockdata
	policy    *isaac.LocalPolicy
	nodepool  *network.Nodepool
	suffrage  base.Suffrage
	states    states.States
	network   network.Server
	sealCache cache.Cache
	logger    *zerolog.Logger
	encs      *encoder.Encoders
}

func SettingNetworkHandlersFromContext(ctx context.Context) (*SettingNetworkHandlers, error) {
	sn := &SettingNetworkHandlers{}

	if err := sn.loadFromContext(ctx); err != nil {
		return nil, err
	}

	return sn, nil
}

func (sn *SettingNetworkHandlers) loadFromContext(ctx context.Context) error {
	if err := LoadVersionContextValue(ctx, &sn.version); err != nil {
		return err
	}
	if err := config.LoadConfigContextValue(ctx, &sn.conf); err != nil {
		return err
	}
	if err := LoadPolicyContextValue(ctx, &sn.policy); err != nil {
		return err
	}
	if err := LoadNodepoolContextValue(ctx, &sn.nodepool); err != nil {
		return err
	}
	if err := LoadDatabaseContextValue(ctx, &sn.database); err != nil {
		return err
	}
	if err := LoadBlockdataContextValue(ctx, &sn.blockdata); err != nil {
		return err
	}
	if err := LoadSuffrageContextValue(ctx, &sn.suffrage); err != nil {
		return err
	}
	if err := LoadConsensusStatesContextValue(ctx, &sn.states); err != nil {
		return err
	}
	if err := config.LoadEncodersContextValue(ctx, &sn.encs); err != nil {
		return err
	}

	i, err := cache.NewCacheFromURI(sn.conf.Network().SealCache().String())
	if err != nil {
		return err
	}
	sn.sealCache = i

	l := zerolog.Nop()
	sn.logger = &l
	if err := LoadNetworkContextValue(ctx, &sn.network); err != nil {
		return err
	}

	if l, ok := sn.network.(logging.HasLogger); ok {
		sn.logger = l.Log()
	}

	return nil
}

func (sn *SettingNetworkHandlers) Set() error {
	sn.network.SetGetStagedOperationsHandler(sn.handlerGetStagedOperations())
	sn.network.SetNewSealHandler(sn.handlerNewSeal())
	sn.network.SetNodeInfoHandler(sn.handlerNodeInfo())
	sn.network.SetBlockdataMapsHandler(sn.handlerBlockdataMaps())
	sn.network.SetBlockdataHandler(sn.handlerBlockdata())
	sn.network.SetStartHandoverHandler(sn.handlerStartHandover())
	sn.network.SetPingHandoverHandler(sn.handlerPingHandover())
	sn.network.SetEndHandoverHandler(sn.handlerEndHandover())
	sn.network.SetGetProposalHandler(sn.handlerGetProposal())

	lc := sn.nodepool.LocalChannel().(*network.DummyChannel)
	lc.SetNewSealHandler(sn.handlerNewSeal())
	lc.SetGetStagedOperationsHandler(sn.handlerGetStagedOperations())
	lc.SetNodeInfoHandler(sn.handlerNodeInfo())
	lc.SetBlockdataMapsHandler(sn.handlerBlockdataMaps())
	lc.SetBlockdataHandler(sn.handlerBlockdata())

	sn.logger.Debug().Msg("local channel handlers binded")

	return nil
}

func (sn *SettingNetworkHandlers) handlerGetStagedOperations() network.GetStagedOperationsHandler {
	return func(hs []valuehash.Hash) ([]operation.Operation, error) {
		return sn.database.StagedOperationsByFact(hs)
	}
}

func (sn *SettingNetworkHandlers) handlerNewSeal() network.NewSealHandler {
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

		if t, ok := sl.(base.Ballot); ok {
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

func (sn *SettingNetworkHandlers) handlerNodeInfo() network.NodeInfoHandler {
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
			sn.policy.Config(),
			nodes,
			sn.suffrage,
			sn.conf.Network().ConnInfo(),
		), nil
	}
}

func (sn *SettingNetworkHandlers) handlerBlockdataMaps() network.BlockdataMapsHandler {
	return func(heights []base.Height) ([]block.BlockdataMap, error) {
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

		maps := make([]block.BlockdataMap, len(filtered))
		for i := range filtered {
			switch m, found, err := sn.database.BlockdataMap(filtered[i]); {
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

func (sn *SettingNetworkHandlers) handlerBlockdata() network.BlockdataHandler {
	return func(p string) (io.Reader, func() error, error) {
		i, err := sn.blockdata.FS().Open(p)
		if err != nil {
			return nil, func() error { return nil }, err
		}
		return i, i.Close, nil
	}
}

func (sn *SettingNetworkHandlers) checkHandoverSeal(sl network.HandoverSeal) error {
	if err := network.IsValidHandoverSeal(
		sn.nodepool.LocalNode(),
		sl,
		sn.policy.NetworkID(),
	); err != nil {
		return err
	}

	ci := sl.ConnInfo()
	if ci.Hint().Type() != network.HTTPConnInfoType {
		return errors.Errorf("only HTTPConnInfoType for handover allowed, not %v", ci.Hint().Type())
	}

	return nil
}

func (sn *SettingNetworkHandlers) handlerStartHandover() network.StartHandoverHandler {
	return func(sl network.StartHandoverSeal) (bool, error) {
		if err := sn.checkHandoverSeal(sl); err != nil {
			return false, network.HandoverRejectedError.Wrap(err)
		}

		if !sl.ConnInfo().Equal(sn.conf.Network().ConnInfo()) {
			return false, network.HandoverRejectedError.Errorf("conninfo not matched")
		}

		if err := sn.states.StartHandover(); err != nil {
			return false, err
		}

		return true, nil
	}
}

func (sn *SettingNetworkHandlers) handlerPingHandover() network.PingHandoverHandler {
	return func(sl network.PingHandoverSeal) (bool, error) {
		if err := sn.checkHandoverSeal(sl); err != nil {
			return false, network.HandoverRejectedError.Wrap(err)
		}

		ch, err := discovery.LoadNodeChannel(sl.ConnInfo(), sn.encs, sn.policy.NetworkConnectionTimeout())
		if err != nil {
			return false, network.HandoverRejectedError.Errorf("failed to load channel from PingHandoverSeal: %w", err)
		}

		if err := sn.nodepool.SetPassthrough(ch, nil, states.DefaultPassthroughExpire); err != nil {
			return false, network.HandoverRejectedError.Errorf("failed to set passthrough from PingHandoverSeal: %w", err)
		}

		return true, nil
	}
}

func (sn *SettingNetworkHandlers) handlerEndHandover() network.EndHandoverHandler {
	return func(sl network.EndHandoverSeal) (bool, error) {
		if err := sn.checkHandoverSeal(sl); err != nil {
			return false, network.HandoverRejectedError.Wrap(err)
		}

		if err := sn.states.EndHandover(sl.ConnInfo()); err != nil {
			return false, err
		}

		return true, nil
	}
}

func (sn *SettingNetworkHandlers) handlerGetProposal() network.GetProposalHandler {
	return func(h valuehash.Hash) (base.Proposal, error) {
		pr, found, err := sn.database.Proposal(h)
		switch {
		case err != nil:
			return nil, err
		case !found:
			return nil, util.NotFoundError.Errorf("proposal not found")
		default:
			return pr, nil
		}
	}
}
