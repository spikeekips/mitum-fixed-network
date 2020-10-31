package launcher

import (
	"fmt"
	"sort"
	"sync"
	"syscall"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Launcher struct {
	sync.RWMutex
	*logging.Logging
	encs            *encoder.Encoders
	design          *NodeDesign
	version         util.Version
	local           *isaac.Local
	consensusStates *isaac.ConsensusStates
	networkID       base.NetworkID

	storage           storage.Storage
	network           network.Server
	nodeChannel       network.Channel
	suffrage          base.Suffrage
	proposalProcessor isaac.ProposalProcessor
	publishURL        string
	sealCache         cache.Cache
}

func NewLauncher(design *NodeDesign, version util.Version) (*Launcher, error) {
	var encs *encoder.Encoders
	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonenc.NewEncoder(), bsonenc.NewEncoder()},
	); err != nil {
		return nil, xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		encs = e
	}

	var sealCache cache.Cache
	if ca, err := cache.NewGCache("lru", 100*100, time.Minute*3); err != nil {
		return nil, err
	} else {
		sealCache = ca
	}

	bn := &Launcher{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "base-node-runner")
		}),
		encs:      encs,
		design:    design,
		version:   version,
		sealCache: sealCache,
	}

	return bn.SetLocal(design.Address(), design.Privatekey(), design.NetworkID())
}

func (bn *Launcher) Encoders() *encoder.Encoders {
	return bn.encs
}

func (bn *Launcher) AddHinters(hinters ...hint.Hinter) error {
	for _, h := range hinters {
		if err := bn.encs.AddHinter(h); err != nil {
			return err
		}
	}

	return nil
}

func (bn *Launcher) SetLocal(
	address base.Address,
	privateKey key.Privatekey,
	networkID base.NetworkID,
) (*Launcher, error) {
	node := isaac.NewLocalNode(address, privateKey, bn.design.Network.PublishURL().String())

	var enc *jsonenc.Encoder
	if e, err := bn.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, err
	} else {
		enc = e.(*jsonenc.Encoder)
	}

	if bn.design.Config.IsDev {
		syscall.Umask(0)
		localfs.DefaultFilePermission = 0o666
		localfs.DefaultDirectoryPermission = 0o777
	}

	var blockfs *storage.BlockFS
	if fs, err := localfs.NewFS(bn.design.BlockFS, true); err != nil {
		return nil, err
	} else {
		blockfs = storage.NewBlockFS(fs, enc)
	}

	if ls, err := isaac.NewLocal(bn.storage, blockfs, node, networkID); err != nil {
		return nil, err
	} else {
		bn.local = ls
		bn.networkID = networkID

		return bn, nil
	}
}

func (bn *Launcher) reloadLocal() error {
	_ = bn.local.SetStorage(bn.storage)

	if err := bn.local.Initialize(); err != nil {
		return err
	}

	co := bn.design.Config
	lpo := bn.local.Policy()

	if _, err := lpo.SetTimeoutWaitingProposal(co.TimeoutWaitingProposal); err != nil {
		return err
	}
	if _, err := lpo.SetIntervalBroadcastingINITBallot(co.IntervalBroadcastingINITBallot); err != nil {
		return err
	}
	if _, err := lpo.SetIntervalBroadcastingProposal(co.IntervalBroadcastingProposal); err != nil {
		return err
	}
	if _, err := lpo.SetWaitBroadcastingACCEPTBallot(co.WaitBroadcastingACCEPTBallot); err != nil {
		return err
	}
	if _, err := lpo.SetIntervalBroadcastingACCEPTBallot(co.IntervalBroadcastingACCEPTBallot); err != nil {
		return err
	}
	if _, err := lpo.SetTimespanValidBallot(co.TimespanValidBallot); err != nil {
		return err
	}
	if _, err := lpo.SetTimeoutProcessProposal(co.TimeoutProcessProposal); err != nil {
		return err
	}

	return nil
}

func (bn *Launcher) Local() *isaac.Local {
	return bn.local
}

func (bn *Launcher) SetStorage(st storage.Storage) *Launcher {
	bn.storage = st

	return bn
}

func (bn *Launcher) Storage() storage.Storage {
	return bn.storage
}

func (bn *Launcher) SetNetwork(nt network.Server) *Launcher {
	bn.network = nt

	bn.network.SetHasSealHandler(bn.networkHandlerHasSeal)
	bn.network.SetGetSealsHandler(bn.networkHandlerGetSeals)
	bn.network.SetNewSealHandler(bn.networkhandlerNewSeal)
	bn.network.SetGetManifestsHandler(bn.networkhandlerGetManifests)
	bn.network.SetGetBlocksHandler(bn.networkhandlerGetBlocks)
	bn.network.SetNodeInfoHandler(bn.NodeInfo)

	return bn
}

func (bn *Launcher) Network() network.Server {
	return bn.network
}

func (bn *Launcher) SetNodeChannel(nc network.Channel) *Launcher {
	bn.nodeChannel = nc

	return bn
}

func (bn *Launcher) NodeChannel() network.Channel {
	return bn.nodeChannel
}

func (bn *Launcher) SetSuffrage(sf base.Suffrage) *Launcher {
	bn.suffrage = sf

	return bn
}

func (bn *Launcher) Suffrage() base.Suffrage {
	return bn.suffrage
}

func (bn *Launcher) SetProposalProcessor(pp isaac.ProposalProcessor) *Launcher {
	bn.proposalProcessor = pp

	return bn
}

func (bn *Launcher) SetPublichURL(s string) *Launcher {
	bn.publishURL = s

	return bn
}

func (bn *Launcher) ProposalProcessor() isaac.ProposalProcessor {
	return bn.proposalProcessor
}

func (bn *Launcher) Initialize() error {
	bn.Log().Info().Msg("trying to initialize")

	components := [][2]interface{}{
		{"storage", bn.storage},
		{"network", bn.network},
		{"nodeChannel", bn.nodeChannel},
		{"reload-local", bn.reloadLocal},
		{"suffrage", bn.suffrage},
		{"proposalProcessor", bn.proposalProcessor},
	}
	for i := range components {
		if err := bn.initialize(components[i]); err != nil {
			return err
		}
	}

	bn.local.Nodes().Traverse(func(n network.Node) bool {
		if l, ok := n.(logging.SetLogger); ok {
			_ = l.SetLogger(bn.Log())
		}

		return true
	})

	if err := bn.createConsensusStates(); err != nil {
		return err
	}

	bn.Log().Info().Msg("all initialized")

	return nil
}

func (bn *Launcher) initialize(i [2]interface{}) error {
	bn.Log().Info().Msg("trying to initialize")

	var name string
	if n, ok := i[0].(string); !ok {
		return xerrors.Errorf("can not initialize component; name not found")
	} else {
		name = n
	}

	var component interface{}
	var initialize func() error
	switch t := i[1].(type) {
	case util.Initializer:
		initialize = t.Initialize
		component = t
	case func() error:
		initialize = t
	}

	if initialize == nil {
		if n, err := bn.createDefaultComponent(name); err != nil {
			return xerrors.Errorf("failed to create default: %w", err)
		} else {
			component = n
			initialize = n.(util.Initializer).Initialize
		}
	}

	if err := initialize(); err != nil {
		return xerrors.Errorf("failed to initialize for %s: %w", name, err)
	}

	if component != nil {
		if l, ok := component.(logging.SetLogger); ok {
			_ = l.SetLogger(bn.Log())
		}
	}

	bn.Log().Info().Str("target", name).Str("target_component", fmt.Sprintf("%T", component)).Msg("initialized")

	return nil
}

func (bn *Launcher) Start() error {
	bn.Log().Info().Msg("trying to start")

	if bn.network == nil {
		return xerrors.Errorf("network is empty")
	}

	if err := bn.network.Start(); err != nil {
		return err
	}

	bn.Log().Info().Interface("policy", bn.local.Policy()).Msg("policies")

	if err := bn.consensusStates.Start(); err != nil {
		return err
	}

	bn.Log().Info().Msg("started")

	return nil
}

func (bn *Launcher) ErrChan() <-chan error {
	return bn.consensusStates.ErrChan()
}

func (bn *Launcher) ConsensusStates() *isaac.ConsensusStates {
	return bn.consensusStates
}

func (bn *Launcher) createConsensusStates() error {
	proposalMaker := isaac.NewProposalMaker(bn.local)

	var booting, joining, consensus, syncing, broken isaac.StateHandler
	var err error
	if booting, err = isaac.NewStateBootingHandler(bn.local, bn.suffrage); err != nil {
		return err
	}
	syncing = isaac.NewStateSyncingHandler(bn.local)
	if joining, err = isaac.NewStateJoiningHandler(bn.local, bn.proposalProcessor); err != nil {
		return err
	}
	if consensus, err = isaac.NewStateConsensusHandler(
		bn.local, bn.proposalProcessor, bn.suffrage, proposalMaker,
	); err != nil {
		return err
	}
	if broken, err = isaac.NewStateBrokenHandler(bn.local); err != nil {
		return err
	}
	for _, h := range []interface{}{booting, joining, consensus, syncing, broken} {
		if l, ok := h.(logging.SetLogger); ok {
			_ = l.SetLogger(bn.Log())
		}
	}

	ballotbox := isaac.NewBallotbox(
		func() []base.Address {
			return bn.suffrage.Nodes()
		},
		func() base.Threshold {
			if t, err := base.NewThreshold(
				uint(len(bn.suffrage.Nodes())),
				bn.local.Policy().ThresholdRatio(),
			); err != nil {
				panic(err)
			} else {
				return t
			}
		},
	)
	_ = ballotbox.SetLogger(bn.Log())

	if cs, err := isaac.NewConsensusStates(
		bn.local, ballotbox, bn.suffrage,
		booting.(*isaac.StateBootingHandler),
		joining.(*isaac.StateJoiningHandler),
		consensus.(*isaac.StateConsensusHandler),
		syncing.(*isaac.StateSyncingHandler),
		broken.(*isaac.StateBrokenHandler),
	); err != nil {
		return err
	} else {
		_ = cs.SetLogger(bn.Log())
		bn.consensusStates = cs

		return nil
	}
}

func (bn *Launcher) networkHandlerHasSeal(h valuehash.Hash) (bool, error) {
	return bn.storage.HasSeal(h)
}

func (bn *Launcher) networkHandlerGetSeals(hs []valuehash.Hash) ([]seal.Seal, error) {
	var sls []seal.Seal

	if err := bn.storage.SealsByHash(hs, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
		sls = append(sls, sl)

		return true, nil
	}, true); err != nil {
		return nil, err
	}

	return sls, nil
}

func (bn *Launcher) networkhandlerNewSeal(sl seal.Seal) error {
	sealChecker := isaac.NewSealValidationChecker(
		sl,
		bn.storage,
		bn.local.Policy(),
		bn.sealCache,
	)
	if err := util.NewChecker("network-new-seal-checker", []util.CheckerFunc{
		sealChecker.CheckIsKnown,
		sealChecker.CheckIsValid,
		func() (bool, error) {
			// NOTE stores seal regardless further checkings.
			if err := bn.storage.NewSeals([]seal.Seal{sl}); err != nil {
				if !xerrors.Is(err, storage.DuplicatedError) {
					return false, err
				}
			}

			return true, nil
		},
	}).Check(); err != nil {
		if xerrors.Is(err, util.CheckerNilError) {
			bn.Log().Debug().Msg(err.Error())

			return nil
		}

		return err
	}

	if t, ok := sl.(ballot.Ballot); ok {
		if checker, err := isaac.NewBallotChecker(t, bn.local, bn.suffrage); err != nil {
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

	bn.consensusStates.NewSeal(sl)

	return nil
}

func (bn *Launcher) networkhandlerGetManifests(heights []base.Height) ([]block.Manifest, error) {
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

		switch m, found, err := bn.storage.ManifestByHeight(h); {
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

func (bn *Launcher) networkhandlerGetBlocks(heights []base.Height) ([]block.Block, error) {
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	var blocks []block.Block
	for _, h := range heights {
		if blk, err := bn.local.BlockFS().Load(h); err != nil {
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

func (bn *Launcher) NodeInfo() (network.NodeInfo, error) {
	var state base.State = base.StateUnknown
	if handler := bn.consensusStates.ActiveHandler(); handler != nil {
		state = handler.State()
	}

	var manifest block.Manifest
	if m, found, err := bn.storage.LastManifest(); err != nil {
		return nil, err
	} else if found {
		manifest = m
	}

	suffrage := make([]base.Node, bn.local.Nodes().Len())
	var i int
	bn.local.Nodes().Traverse(func(n network.Node) bool {
		suffrage[i] = n
		i++

		return true
	})

	return network.NewNodeInfoV0(
		bn.local.Node(),
		bn.local.Policy().NetworkID(),
		state,
		manifest,
		bn.version,
		bn.publishURL,
		bn.local.Policy().Policy(),
		bn.local.Policy().Config(),
		suffrage,
	), nil
}

func (bn *Launcher) createDefaultComponent(name string) (interface{}, error) {
	switch name {
	case "suffrage":
		if err := bn.DefaultSuffrage(); err != nil {
			return nil, err
		}
		return bn.suffrage, nil
	case "proposalProcessor":
		if err := bn.DefaultProposalProcessor(); err != nil {
			return nil, err
		}

		return bn.proposalProcessor, nil
	default:
		return nil, xerrors.Errorf("this component, %s should be set manually", name)
	}
}

func (bn *Launcher) DefaultSuffrage() error {
	l := bn.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "default-suffrage")
	})
	l.Debug().Msg("trying to attach")

	bn.suffrage = NewRoundrobinSuffrage(bn.local, 100)

	l.Debug().Msg("attached")

	return nil
}

func (bn *Launcher) DefaultProposalProcessor() error {
	l := bn.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "default-proposal-processor")
	})
	l.Debug().Msg("trying to attach")

	bn.proposalProcessor = isaac.NewDefaultProposalProcessor(bn.local, bn.suffrage)

	l.Debug().Msg("attached")

	return nil
}
