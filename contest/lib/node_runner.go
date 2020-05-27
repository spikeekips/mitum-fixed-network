package contestlib

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type NodeRunner struct {
	*logging.Logging
	design            *NodeDesign
	encs              *encoder.Encoders
	je                encoder.Encoder
	localstate        *isaac.Localstate
	storage           storage.Storage
	network           network.Server
	ballotbox         *isaac.Ballotbox
	suffrage          base.Suffrage
	proposalProcessor isaac.ProposalProcessor
	proposalMaker     *isaac.ProposalMaker
	consensusStates   *isaac.ConsensusStates
}

func NewNodeRunnerFromDesign(design *NodeDesign, encs *encoder.Encoders) (*NodeRunner, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonencoder.JSONType, ""); err != nil { // NOTE get latest bson encoder
		return nil, xerrors.Errorf("json encoder needs for quic-network: %w", err)
	} else {
		je = e
	}

	return &NodeRunner{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "contest-node-runner")
		}),
		design: design,
		encs:   encs,
		je:     je,
	}, nil
}

func (nr *NodeRunner) Design() *NodeDesign {
	return nr.design
}

func (nr *NodeRunner) Localstate() *isaac.Localstate {
	return nr.localstate
}

func (nr *NodeRunner) Storage() storage.Storage {
	return nr.storage
}

func (nr *NodeRunner) Initialize() error {
	for _, f := range []func() error{
		nr.attachStorage,
		nr.attachLocalstate,
		nr.attachNodeChannel,
		nr.attachRemoteNodes,
		nr.attachBallotbox,
		nr.attachSuffrage,
		nr.attachProposalProcessor,
		nr.attachProposalMaker,
		nr.attachNetwork,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (nr *NodeRunner) attachLocalstate() error {
	var localnode *isaac.LocalNode
	if ca, err := NewContestAddress(nr.design.Address); err != nil {
		return err
	} else {
		localnode = isaac.NewLocalNode(ca, nr.design.Privatekey())
	}

	if ls, err := isaac.NewLocalstate(nr.storage, localnode, nr.design.NetworkID()); err != nil {
		return err
	} else {
		nr.localstate = ls

		return nil
	}
}

func (nr *NodeRunner) attachStorage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "storage")
	})
	l.Debug().Msg("trying to attach")

	if st, err := LoadStorage(nr.design.Storage, nr.encs); err != nil {
		return err
	} else {
		nr.storage = st
	}

	nr.setupLogging(nr.storage)

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachNetwork() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "network")
	})
	l.Debug().Msg("trying to attach")

	nd := nr.design.Network

	var certs []tls.Certificate
	if priv, err := util.GenerateED25519Privatekey(); err != nil {
		return err
	} else if ct, err := util.GenerateTLSCerts(nd.PublishURL().Host, priv); err != nil {
		return err
	} else {
		certs = ct
	}

	var nt network.Server
	if qs, err := quicnetwork.NewPrimitiveQuicServer(nd.Bind, certs); err != nil {
		return err
	} else if nqs, err := quicnetwork.NewQuicServer(qs, nr.encs, nr.je); err != nil {
		return err
	} else {
		nt = nqs
	}

	nr.setupLogging(nt)
	nr.network = nt

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachNetworkHandlers() error {
	nr.network.SetGetSealsHandler(nr.networkHandlerGetSeal)
	nr.network.SetNewSealHandler(nr.networkhandlerNewSeal)
	nr.network.SetGetManifests(nr.networkhandlerGetManifests)
	nr.network.SetGetBlocks(nr.networkhandlerGetBlocks)

	return nil
}

func (nr *NodeRunner) networkHandlerGetSeal(hs []valuehash.Hash) ([]seal.Seal, error) {
	var sls []seal.Seal
	for _, h := range hs {
		if sl, err := nr.storage.Seal(h); err != nil {
			if !xerrors.Is(err, storage.NotFoundError) {
				continue
			}

			return nil, err
		} else {
			sls = append(sls, sl)
		}
	}

	return sls, nil
}

func (nr *NodeRunner) networkhandlerNewSeal(sl seal.Seal) error {
	sealChecker := isaac.NewSealValidationChecker(sl, nr.localstate.Policy().NetworkID())
	if err := util.NewChecker("network-new-seal-checker", []util.CheckerFunc{
		sealChecker.CheckIsValid,
	}).Check(); err != nil {
		return err
	}

	// NOTE stores seal regardless further checkings.
	if err := nr.storage.NewSeals([]seal.Seal{sl}); err != nil {
		return err
	}

	if t, ok := sl.(ballot.Ballot); ok {
		if checker, err := isaac.NewBallotChecker(t, nr.localstate, nr.suffrage); err != nil {
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

	if err := nr.consensusStates.NewSeal(sl); err != nil {
		nr.Log().Error().Err(err).Msg("failed to receive seal by consensus states")

		return err
	}

	return nil
}

func (nr *NodeRunner) networkhandlerGetManifests(heights []base.Height) ([]block.Manifest, error) {
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

		if m, err := nr.storage.ManifestByHeight(h); err != nil {
			if !xerrors.Is(err, storage.NotFoundError) {
				return nil, err
			}
		} else {
			manifests = append(manifests, m)
		}
	}

	return manifests, nil
}

func (nr *NodeRunner) networkhandlerGetBlocks(heights []base.Height) ([]block.Block, error) {
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	var blocks []block.Block
	fetched := map[base.Height]struct{}{}
	for _, h := range heights {
		if _, found := fetched[h]; found {
			continue
		}

		fetched[h] = struct{}{}

		if m, err := nr.storage.BlockByHeight(h); err != nil {
			if !xerrors.Is(err, storage.NotFoundError) {
				return nil, err
			}
		} else {
			blocks = append(blocks, m)
		}
	}

	return blocks, nil
}

func (nr *NodeRunner) attachNodeChannel() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "node-channel")
	})
	l.Debug().Msg("trying to attach")

	nu := new(url.URL)
	*nu = *nr.design.Network.PublishURL()
	nu.Host = fmt.Sprintf("localhost:%s", nu.Port())

	var channel network.NetworkChannel
	if ch, err := createNodeChannel(nu, nr.encs, nr.je); err != nil {
		return err
	} else {
		channel = ch
	}

	nr.setupLogging(channel)
	_ = nr.localstate.Node().SetChannel(channel)

	l.Debug().Msg("attached")

	return nil
}

func createNodeChannel(publish *url.URL, encs *encoder.Encoders, enc encoder.Encoder) (network.NetworkChannel, error) {
	var channel network.NetworkChannel

	nu := new(url.URL)
	*nu = *publish
	nu.Scheme = "https"

	switch publish.Scheme {
	case "quic":
		if ch, err := quicnetwork.NewQuicChannel(
			nu.String(),
			100,
			true,
			time.Second*1,
			3,
			nil,
			encs,
			enc,
		); err != nil {
			return nil, err
		} else {
			channel = ch
		}
	default:
		return nil, xerrors.Errorf("unsupported publish URL, %v", publish.String())
	}

	return channel, nil
}

func (nr *NodeRunner) attachRemoteNodes() error {
	nodes := make([]isaac.Node, len(nr.design.Nodes))

	for i, r := range nr.design.Nodes {
		r := r
		l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Str("address", r.Address)
		})

		l.Debug().Msg("trying to create remote node")

		var n *isaac.RemoteNode
		if ca, err := NewContestAddress(r.Address); err != nil {
			return err
		} else {
			n = isaac.NewRemoteNode(ca, r.Publickey())
		}

		if ch, err := createNodeChannel(r.NetworkURL(), nr.encs, nr.je); err != nil {
			return err
		} else {
			nr.setupLogging(ch)

			_ = n.SetChannel(ch)
		}
		l.Debug().Msg("created")

		nodes[i] = n
	}

	if err := nr.localstate.Nodes().Add(nodes...); err != nil {
		return err
	}

	return nil
}

func (nr *NodeRunner) attachBallotbox() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "ballotbox")
	})
	l.Debug().Msg("trying to attach")

	bb := isaac.NewBallotbox(func() base.Threshold {
		return nr.localstate.Policy().Threshold()
	})

	nr.setupLogging(bb)
	nr.ballotbox = bb

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachSuffrage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "suffrage")
	})
	l.Debug().Msg("trying to attach")

	var sf base.Suffrage
	if s, err := nr.design.Component.Suffrage.New(nr.localstate); err != nil {
		return xerrors.Errorf("failed to create new suffrage component: %w", err)
	} else {
		sf = s
	}

	nr.setupLogging(sf)
	nr.suffrage = sf

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachProposalProcessor() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "proposal-processor")
	})
	l.Debug().Msg("trying to attach")

	pp := isaac.NewProposalProcessorV0(nr.localstate)

	nr.setupLogging(pp)
	nr.proposalProcessor = pp

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachProposalMaker() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "proposal-maker")
	})
	l.Debug().Msg("trying to attach")

	pm := isaac.NewProposalMaker(nr.localstate)

	nr.setupLogging(pm)
	nr.proposalMaker = pm

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachConsensusStates() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "consensus-states")
	})
	l.Debug().Msg("trying to attach")

	var booting, joining, consensus, syncing, broken isaac.StateHandler
	{
		l.Debug().Str("state", "booting").Msg("trying to attach")
		var err error
		if booting, err = isaac.NewStateBootingHandler(nr.localstate, nr.proposalProcessor); err != nil {
			return err
		}
		l.Debug().Str("state", "syncing").Msg("trying to attach")
		if syncing, err = isaac.NewStateSyncingHandler(nr.localstate, nr.proposalProcessor); err != nil {
			return err
		}
		l.Debug().Str("state", "joining").Msg("trying to attach")
		if joining, err = isaac.NewStateJoiningHandler(nr.localstate, nr.proposalProcessor); err != nil {
			return err
		}
		l.Debug().Str("state", "consensus").Msg("trying to attach")
		if consensus, err = isaac.NewStateConsensusHandler(
			nr.localstate, nr.proposalProcessor, nr.suffrage, nr.proposalMaker,
		); err != nil {
			return err
		}
		l.Debug().Str("state", "broken").Msg("trying to attach")
		if broken, err = isaac.NewStateBrokenHandler(nr.localstate); err != nil {
			return err
		}
	}
	for _, h := range []interface{}{booting, joining, consensus, syncing, broken} {
		nr.setupLogging(h)
	}

	ss := isaac.NewConsensusStates(
		nr.localstate,
		nr.ballotbox,
		nr.suffrage,
		booting.(*isaac.StateBootingHandler),
		joining.(*isaac.StateJoiningHandler),
		consensus.(*isaac.StateConsensusHandler),
		syncing.(*isaac.StateSyncingHandler),
		broken.(*isaac.StateBrokenHandler),
	)

	nr.setupLogging(ss)

	nr.consensusStates = ss

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) Start() error {
	nr.Log().Info().Msg("NodeRunner trying to start")

	for _, f := range []func() error{
		nr.attachConsensusStates,
		nr.attachNetworkHandlers,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	if err := nr.network.Start(); err != nil {
		return err
	}

	if err := nr.consensusStates.Start(); err != nil {
		return err
	}

	nr.Log().Info().Msg("NodeRunner started")

	return nil
}

func (nr *NodeRunner) setupLogging(i interface{}) {
	if l, ok := i.(logging.SetLogger); ok {
		_ = l.SetLogger(nr.Log())
	}
}

func parseDurationFromQuery(query url.Values, key string, v time.Duration) (time.Duration, error) {
	if sl, found := query[key]; !found || len(sl) < 1 {
		return v, nil
	} else if s := sl[len(sl)-1]; len(strings.TrimSpace(s)) < 1 { // pop last one
		return v, nil
	} else if d, err := time.ParseDuration(s); err != nil {
		return 0, xerrors.Errorf("invalid %s value for mongodb: %w", key, err)
	} else {
		return d, nil
	}
}

func newMongodbStorage(uri string, encs *encoder.Encoders) (storage.Storage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, xerrors.Errorf("invalid storge uri: %w", err)
	}

	connectTimeout := time.Second * 2
	execTimeout := time.Second * 2
	{
		query := parsed.Query()
		if d, err := parseDurationFromQuery(query, "connectTimeout", connectTimeout); err != nil {
			return nil, err
		} else {
			connectTimeout = d
		}
		if d, err := parseDurationFromQuery(query, "execTimeout", execTimeout); err != nil {
			return nil, err
		} else {
			execTimeout = d
		}
	}

	var be encoder.Encoder
	if e, err := encs.Encoder(bsonencoder.BSONType, ""); err != nil { // NOTE get latest bson encoder
		return nil, xerrors.Errorf("bson encoder needs for mongodb: %w", err)
	} else {
		be = e
	}

	if client, err := mongodbstorage.NewClient(uri, connectTimeout, execTimeout); err != nil {
		return nil, err
	} else if st, err := mongodbstorage.NewStorage(client, encs, be); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}
