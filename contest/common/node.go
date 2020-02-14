package common

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/seal"
)

var (
	ContestAddressType hint.Type = hint.Type([2]byte{0xd0, 0x00})
	ContestAddressHint hint.Hint = hint.MustHint(ContestAddressType, "0.1")
)

func NewLocalNode(id int) *isaac.LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := isaac.NewLocalNode(NewContestAddress(id), pk)

	channel := network.NewChanChannel(0, nil)

	return ln.SetChannel(channel)
}

func NewNode(id int) *isaac.LocalState {
	// create new node
	localNode := NewLocalNode(id)
	localState := isaac.NewLocalState(localNode, isaac.NewLocalPolicy())

	// NOTE only one node does not use SealHandler
	localNode.Channel().(*network.ChanChannel).SetSealHandler(func(sl seal.Seal) (seal.Seal, error) {
		return sl, nil
	})

	return localState
}

type NodeProcess struct {
	*logging.Logger
	LocalState        *isaac.LocalState
	Ballotbox         *isaac.Ballotbox
	Suffrage          isaac.Suffrage
	SealStorage       isaac.SealStorage
	ProposalProcessor isaac.ProposalProcessor
	ConsensusStates   *isaac.ConsensusStates
	AllNodes          []*isaac.LocalState
	stopChan          chan struct{}
}

func NewNodeProcess(localState *isaac.LocalState, initialBlock isaac.Block) (*NodeProcess, error) {
	_ = localState.SetLastBlock(initialBlock)

	ballotbox := isaac.NewBallotbox(localState)
	suffrage := NewRoundrobinSuffrage(localState, 100)
	sealStorage := NewMapSealStorage()
	proposalProcessor := isaac.NewProposalProcessorV0(localState, sealStorage)

	cshandlerBooting, err := isaac.NewConsensusStateBootingHandler(localState)
	if err != nil {
		return nil, err
	}

	cshandlerJoining, err := isaac.NewConsensusStateJoiningHandler(localState, proposalProcessor)
	if err != nil {
		return nil, err
	}

	proposalMaker := isaac.NewProposalMaker(localState)
	cshandlerConsensus, err := isaac.NewConsensusStateConsensusHandler(
		localState,
		proposalProcessor,
		suffrage,
		proposalMaker,
	)
	if err != nil {
		return nil, err
	}

	css := isaac.NewConsensusStates(
		localState,
		ballotbox,
		suffrage,
		sealStorage,
		cshandlerBooting,
		cshandlerJoining,
		cshandlerConsensus,
		nil,
		nil,
	)

	return &NodeProcess{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c
		}),
		LocalState:        localState,
		Ballotbox:         ballotbox,
		Suffrage:          suffrage,
		SealStorage:       sealStorage,
		ProposalProcessor: proposalProcessor,
		ConsensusStates:   css,
		stopChan:          make(chan struct{}, 2),
	}, nil
}

func (np *NodeProcess) Start() error {
	go func() {
	end:
		for {
			select {
			case <-np.stopChan:
				break end
			case sl := <-np.LocalState.Node().Channel().ReceiveSeal():
				go func() {
					if err := np.ConsensusStates.NewSeal(sl); err != nil {
						np.Log().Error().Err(err).Msg("ConsensusStates failed to receive seal")
					}
				}()
			}
		}
	}()

	return np.ConsensusStates.Start()
}

func (np *NodeProcess) Stop() error {
	np.stopChan <- struct{}{}
	return np.ConsensusStates.Stop()
}

func (np *NodeProcess) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = np.Logger.SetLogger(l)

	np.setLogger(np.ConsensusStates, l)
	np.setLogger(np.ProposalProcessor, l)
	np.setLogger(np.Ballotbox, l)
	np.setLogger(np.Suffrage, l)
	np.setLogger(np.SealStorage, l)

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
