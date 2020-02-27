package common

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

var ports []int

func NewLocalNode(id int) *isaac.LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := isaac.NewLocalNode(NewContestAddress(id), pk)

	return ln
}

func NewNodeChannel(encs *encoder.Encoders, enc encoder.Encoder, netType string) network.Channel {
	var channel network.Channel

	switch netType {
	case "quic":
		port, err := FreePort("udp", ports)
		if err != nil {
			panic(err)
		}

		ch, err := network.NewQuicChannel(
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
		channel = network.NewChanChannel(100000000)
	}

	return channel
}

func NewNode(id int, netType string) (*isaac.Localstate, error) {
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
	db, _ := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	st := isaac.NewLeveldbStorage(db, encs, enc)

	localNode := NewLocalNode(id)
	localstate, err := isaac.NewLocalstate(st, localNode)
	if err != nil {
		return nil, err
	}

	_ = localNode.SetChannel(NewNodeChannel(encs, enc, netType))

	return localstate, nil
}

type NodeProcess struct {
	*logging.Logger
	Localstate        *isaac.Localstate
	Ballotbox         *isaac.Ballotbox
	Suffrage          isaac.Suffrage
	ProposalProcessor isaac.ProposalProcessor
	ConsensusStates   *isaac.ConsensusStates
	NetworkServer     network.Server
	AllNodes          []*isaac.Localstate
	stopChan          chan struct{}
}

func NewSuffrage(localstate *isaac.Localstate) isaac.Suffrage {
	return NewRoundrobinSuffrage(localstate, 100)
}

func NewNodeProcess(localstate *isaac.Localstate) (*NodeProcess, error) {
	ballotbox := isaac.NewBallotbox(func() isaac.Threshold {
		return localstate.Policy().Threshold()
	})
	suffrage := NewSuffrage(localstate)
	proposalProcessor := isaac.NewProposalProcessorV0(localstate)

	cshandlerBooting, err := isaac.NewStateBootingHandler(localstate, proposalProcessor)
	if err != nil {
		return nil, err
	}

	cshandlerJoining, err := isaac.NewStateJoiningHandler(localstate, proposalProcessor)
	if err != nil {
		return nil, err
	}

	proposalMaker := isaac.NewProposalMaker(localstate)
	cshandlerConsensus, err := isaac.NewStateConsensusHandler(
		localstate,
		proposalProcessor,
		suffrage,
		proposalMaker,
	)
	if err != nil {
		return nil, err
	}

	css := isaac.NewConsensusStates(
		localstate,
		ballotbox,
		suffrage,
		cshandlerBooting,
		cshandlerJoining,
		cshandlerConsensus,
		nil,
		nil,
	)

	np := &NodeProcess{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
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
	case *network.ChanChannel:
		server = network.NewChanServer(ch)
	case *network.QuicChannel:
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

		if s, err := network.NewQuicServer(bind, certs, encs, enc); err != nil {
			return nil, err
		} else {
			server = s
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

func (np *NodeProcess) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = np.Logger.SetLogger(l)

	np.setLogger(np.NetworkServer, l)
	np.setLogger(np.Localstate.Node().Channel(), l)
	np.setLogger(np.ConsensusStates, l)
	np.setLogger(np.ProposalProcessor, l)
	np.setLogger(np.Ballotbox, l)
	np.setLogger(np.Suffrage, l)

	return np.Logger
}

func (np *NodeProcess) setLogger(i interface{}, l zerolog.Logger) {
	lo, ok := i.(logging.SetLogger)
	if !ok {
		np.Log().Warn().Str("instance", fmt.Sprintf("%T", i)).Msg("failed to SetLogger")
		return
	}

	_ = lo.SetLogger(l)
}
