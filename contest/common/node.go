package common

import (
	"fmt"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	leveldbstorage "github.com/spikeekips/mitum/storage/leveldb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var ports []int

func NewLocalNode(id int) *isaac.LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := isaac.NewLocalNode(NewContestAddress(id), pk)

	return ln
}

func NewNodeChannel(encs *encoder.Encoders, enc encoder.Encoder, netType string) network.NetworkChannel {
	var channel network.NetworkChannel

	switch netType {
	case "quic":
		port, err := FreePort("udp", ports)
		if err != nil {
			panic(err)
		}

		ch, err := quicnetwork.NewQuicChannel(
			fmt.Sprintf("https://localhost:%d", port),
			100,
			true,
			time.Second*1,
			3,
			nil,
			encs,
			enc,
		)
		if err != nil {
			panic(err)
		}
		channel = ch
	case "chan":
		channel = channetwork.NewNetworkChanChannel(100000000)
	}

	return channel
}

func NewNode(id int, networkID []byte, netType string) (*isaac.Localstate, error) {
	// encoder
	encs := encoder.NewEncoders()
	enc := encoder.NewJSONEncoder()
	if err := encs.AddEncoder(enc); err != nil {
		return nil, err
	}

	for i := range Hinters {
		hinter, ok := Hinters[i][1].(hint.Hinter)
		if !ok {
			return nil, xerrors.Errorf("not hint.Hinter: %T", Hinters[i])
		}

		if err := encs.AddHinter(hinter); err != nil {
			return nil, err
		}
	}

	// create new node
	// TODO select db type by configuration
	st := leveldbstorage.NewMemStorage(encs, enc)

	localNode := NewLocalNode(id)
	localstate, err := isaac.NewLocalstate(st, localNode, networkID)
	if err != nil {
		return nil, err
	}

	_ = localNode.SetChannel(NewNodeChannel(encs, enc, netType))

	return localstate, nil
}

type NodeProcess struct {
	*logging.Logging
	Localstate        *isaac.Localstate
	Ballotbox         *isaac.Ballotbox
	Suffrage          base.Suffrage
	ProposalProcessor isaac.ProposalProcessor
	ConsensusStates   *isaac.ConsensusStates
	NetworkServer     network.Server
	AllNodes          []*isaac.Localstate
	stopChan          chan struct{}
}

func NewSuffrage(localstate *isaac.Localstate) base.Suffrage {
	return NewRoundrobinSuffrage(localstate, 100)
}

func NewNodeProcess(localstate *isaac.Localstate) (*NodeProcess, error) {
	ballotbox := isaac.NewBallotbox(func() base.Threshold {
		return localstate.Policy().Threshold()
	})
	suffrage := NewSuffrage(localstate)
	proposalProcessor := isaac.NewProposalProcessorV0(localstate)
	proposalMaker := isaac.NewProposalMaker(localstate)

	var cshandlerBooting, cshandlerJoining, cshandlerConsensus, cshandlerSyncing, cshandlerBroken isaac.StateHandler
	{
		var err error
		if cshandlerBooting, err = isaac.NewStateBootingHandler(localstate, proposalProcessor); err != nil {
			return nil, err
		}
		if cshandlerSyncing, err = isaac.NewStateSyncingHandler(localstate, proposalProcessor); err != nil {
			return nil, err
		}
		if cshandlerJoining, err = isaac.NewStateJoiningHandler(localstate, proposalProcessor); err != nil {
			return nil, err
		}
		if cshandlerBroken, err = isaac.NewStateBrokenHandler(localstate); err != nil {
			return nil, err
		}
		if cshandlerConsensus, err = isaac.NewStateConsensusHandler(
			localstate, proposalProcessor, suffrage, proposalMaker); err != nil {
			return nil, err
		}
	}

	css := isaac.NewConsensusStates(
		localstate, ballotbox, suffrage,
		cshandlerBooting.(*isaac.StateBootingHandler),
		cshandlerJoining.(*isaac.StateJoiningHandler),
		cshandlerConsensus.(*isaac.StateConsensusHandler),
		cshandlerSyncing.(*isaac.StateSyncingHandler),
		cshandlerBroken.(*isaac.StateBrokenHandler),
	)

	np := &NodeProcess{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c
		}),
		Localstate:        localstate,
		Ballotbox:         ballotbox,
		Suffrage:          suffrage,
		ProposalProcessor: proposalProcessor,
		ConsensusStates:   css,
		stopChan:          make(chan struct{}, 2),
	}

	{
		if server, err := np.networkServer(); err != nil {
			return nil, err
		} else {
			np.NetworkServer = server
		}
	}

	return np, nil
}

func (np *NodeProcess) networkServer() (network.Server, error) {
	var server network.Server
	switch ch := np.Localstate.Node().Channel().(type) {
	case *channetwork.NetworkChanChannel:
		server = channetwork.NewNetworkChanServer(ch)
	case *quicnetwork.QuicChannel:
		priv, err := util.GenerateED25519Privatekey()
		if err != nil {
			return nil, err
		}

		certs, err := util.GenerateTLSCerts(ch.URL().Host, priv)
		if err != nil {
			return nil, err
		}

		bind := ch.URL().Host

		encs := np.Localstate.Storage().Encoders()
		enc, err := encs.Encoder(
			encoder.JSONEncoder{}.Hint().Type(),
			encoder.JSONEncoder{}.Hint().Version(),
		)
		if err != nil {
			return nil, err
		}

		if qs, err := quicnetwork.NewPrimitiveQuicServer(bind, certs); err != nil {
			return nil, err
		} else if nqs, err := quicnetwork.NewQuicServer(qs, encs, enc); err != nil {
			return nil, err
		} else {
			server = nqs
		}
	default:
		return nil, xerrors.Errorf("unknown network found")
	}

	server.SetNewSealHandler(func(sl seal.Seal) error {
		if err := np.ConsensusStates.NewSeal(sl); err != nil {
			np.Log().Error().Err(err).Msg("ConsensusStates failed to receive seal")

			return err
		}

		return nil
	})

	return server, nil
}

func (np *NodeProcess) Start() error {
	if err := np.NetworkServer.Start(); err != nil {
		return err
	}

	return np.ConsensusStates.Start()
}

func (np *NodeProcess) Stop() error {
	np.stopChan <- struct{}{}
	return np.ConsensusStates.Stop()
}

func (np *NodeProcess) SetLogger(l logging.Logger) logging.Logger {
	_ = np.Logging.SetLogger(l)

	np.setLogger(np.NetworkServer, l)
	np.setLogger(np.Localstate.Node().Channel(), l)
	np.setLogger(np.ConsensusStates, l)
	np.setLogger(np.ProposalProcessor, l)
	np.setLogger(np.Ballotbox, l)
	np.setLogger(np.Suffrage, l)

	return np.Log()
}

func (np *NodeProcess) setLogger(i interface{}, l logging.Logger) {
	lo, ok := i.(logging.SetLogger)
	if !ok {
		np.Log().Warn().Str("instance", fmt.Sprintf("%T", i)).Msg("failed to SetLogger")
		return
	}

	_ = lo.SetLogger(l)
}
