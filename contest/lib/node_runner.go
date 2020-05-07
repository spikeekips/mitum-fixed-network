package contestlib

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

type NodeRunner struct {
	*logging.Logging
	design            *NodeDesign
	localstate        *isaac.Localstate
	encs              *encoder.Encoders
	storage           storage.Storage
	network           network.Server
	ballotbox         *isaac.Ballotbox
	suffrage          base.Suffrage
	proposalProcessor isaac.ProposalProcessor
	proposalMaker     *isaac.ProposalMaker
	consensusStates   *isaac.ConsensusStates
}

func NewNodeRunnerFromDesign(design *NodeDesign) *NodeRunner {
	return &NodeRunner{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "contest-node-runner")
		}),
		design: design,
	}
}

func (nr *NodeRunner) Localstate() *isaac.Localstate {
	return nr.localstate
}

func (nr *NodeRunner) Initialize() error {
	for _, f := range []func() error{
		nr.attachEncoder,
		nr.attachStorage,
		nr.attachLocalstate,
		nr.attachNetwork,
		nr.attachNodeChannel,
		nr.attachBallotbox,
		nr.attachSuffrage,
		nr.attachProposalProcessor,
		nr.attachProposalMaker,
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
	} else if pk, err := key.NewBTCPrivatekey(); err != nil {
		return err
	} else {
		localnode = isaac.NewLocalNode(ca, pk)
	}

	if ls, err := isaac.NewLocalstate(nr.storage, localnode, nr.design.NetworkID()); err != nil {
		return err
	} else {
		nr.localstate = ls

		return nil
	}
}

func (nr *NodeRunner) attachEncoder() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "encoders")
	})
	l.Debug().Msg("trying to attach")

	encs := encoder.NewEncoders()
	{
		enc := jsonencoder.NewEncoder()
		if err := encs.AddEncoder(enc); err != nil {
			return err
		}
	}

	{
		enc := bsonencoder.NewEncoder()
		if err := encs.AddEncoder(enc); err != nil {
			return err
		}
	}

	for i := range hinters {
		hinter, ok := hinters[i][1].(hint.Hinter)
		if !ok {
			return xerrors.Errorf("not hint.Hinter: %T", hinters[i])
		}

		if err := encs.AddHinter(hinter); err != nil {
			return err
		}
	}

	nr.encs = encs

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachStorage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "storage")
	})
	l.Debug().Msg("trying to attach")

	parsed, err := url.Parse(nr.design.Storage)
	if err != nil {
		return xerrors.Errorf("invalid storge uri: %w", err)
	}

	var st storage.Storage
	switch strings.ToLower(parsed.Scheme) {
	case "mongodb":
		if s, err := newMongodbStorage(nr.design.Storage, nr.encs); err != nil {
			return err
		} else {
			st = s
		}
	default:
		return xerrors.Errorf("failed to find storage by uri")
	}

	_ = nr.setupLogging(st)
	nr.storage = st

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

	var je encoder.Encoder
	if e, err := nr.encs.Encoder(jsonencoder.JSONType, ""); err != nil { // NOTE get latest bson encoder
		return xerrors.Errorf("json encoder needs for quic-network", err)
	} else {
		je = e
	}

	var nt network.Server
	if qs, err := quicnetwork.NewPrimitiveQuicServer(nd.Bind, certs); err != nil {
		return err
	} else if nqs, err := quicnetwork.NewQuicServer(qs, nr.encs, je); err != nil {
		return err
	} else {
		nt = nqs
	}

	_ = nr.setupLogging(nt)
	nr.network = nt

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachNodeChannel() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "node-channel")
	})
	l.Debug().Msg("trying to attach")

	var channel network.NetworkChannel

	switch nr.design.Network.PublishURL().Scheme {
	case "quic":
		var je encoder.Encoder
		if e, err := nr.encs.Encoder(jsonencoder.JSONType, ""); err != nil { // NOTE get latest bson encoder
			return xerrors.Errorf("json encoder needs for quic-network", err)
		} else {
			je = e
		}

		if ch, err := quicnetwork.NewQuicChannel(
			fmt.Sprintf("https://localhost:%d", nr.design.Network.BindPort()),
			100,
			true,
			time.Second*1,
			3,
			nil,
			nr.encs,
			je,
		); err != nil {
			return err
		} else {
			channel = ch
		}
	}

	_ = nr.setupLogging(channel)
	_ = nr.localstate.Node().SetChannel(channel)

	l.Debug().Msg("attached")

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

	_ = nr.setupLogging(bb)
	nr.ballotbox = bb

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) attachSuffrage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "suffrage")
	})
	l.Debug().Msg("trying to attach")

	sf := NewRoundrobinSuffrage(nr.localstate, 100)

	_ = nr.setupLogging(sf)
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

	_ = nr.setupLogging(pp)
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

	_ = nr.setupLogging(pm)
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
		_ = nr.setupLogging(h)
	}

	nr.consensusStates = isaac.NewConsensusStates(
		nr.localstate,
		nr.ballotbox,
		nr.suffrage,
		booting.(*isaac.StateBootingHandler),
		joining.(*isaac.StateJoiningHandler),
		consensus.(*isaac.StateConsensusHandler),
		syncing.(*isaac.StateSyncingHandler),
		broken.(*isaac.StateBrokenHandler),
	)

	l.Debug().Msg("attached")

	return nil
}

func (nr *NodeRunner) Start() error {
	nr.Log().Info().Msg("NodeRunner trying to start")

	if err := nr.attachConsensusStates(); err != nil {
		return err
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

func (nr *NodeRunner) setupLogging(i interface{}) interface{} {
	if l, ok := i.(logging.SetLogger); ok {
		_ = l.SetLogger(nr.Log())
	}

	return i
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
		return nil, xerrors.Errorf("bson encoder needs for mongodb", err)
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
