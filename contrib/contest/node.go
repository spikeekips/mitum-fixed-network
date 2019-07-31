package main

import (
	"golang.org/x/xerrors"

	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/keypair"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type Node struct {
	homeState *isaac.HomeState
	nt        *contest_module.ChannelNetwork
	sc        *isaac.StateController
}

func NewNode(i uint, globalConfig *Config, config *NodeConfig) (*Node, error) {
	home := NewHome(i)

	lastBlock := config.LastBlock()
	previousBlock := config.Block(lastBlock.Height().Sub(1))
	homeState := isaac.NewHomeState(home, previousBlock).
		SetBlock(lastBlock)

	thr, _ := isaac.NewThreshold(globalConfig.NumberOfNodes(), *config.Policy.Threshold)
	cm := isaac.NewCompiler(homeState, isaac.NewBallotbox(thr))
	nt := contest_module.NewChannelNetwork(
		home,
		func(sl seal.Seal) (seal.Seal, error) {
			return sl, xerrors.Errorf("echo back")
		},
	)
	suffrage := contest_module.NewFixedProposerSuffrage(home, home)
	pv := contest_module.NewDummyProposalValidator()

	var sc *isaac.StateController
	{ // state handlers
		bs := isaac.NewBootingStateHandler(homeState)
		js, err := isaac.NewJoinStateHandler(
			homeState,
			cm,
			nt,
			*config.Policy.IntervalBroadcastINITBallotInJoin,
			*config.Policy.TimeoutWaitVoteResultInJoin,
		)
		if err != nil {
			return nil, err
		}
		cs, err := isaac.NewConsensusStateHandler(
			homeState,
			cm,
			nt,
			suffrage,
			pv,
			*config.Policy.TimeoutWaitBallot,
		)
		if err != nil {
			return nil, err
		}
		ss := isaac.NewStoppedStateHandler()

		sc = isaac.NewStateController(homeState, cm, bs, js, cs, ss)

		go func() {
			for m := range nt.Reader() {
				_ = sc.Write(m)
			}
		}()
	}

	n := &Node{
		homeState: homeState,
		nt:        nt,
		sc:        sc,
	}

	return n, nil
}

func (no *Node) Home() node.Home {
	return no.homeState.Home()
}

func (no *Node) Start() error {
	if err := no.nt.Start(); err != nil {
		return err
	}

	if err := no.sc.Start(); err != nil {
		return err
	}

	return nil
}

func (no *Node) Stop() error {
	if err := no.sc.Stop(); err != nil {
		return err
	}

	if err := no.nt.Stop(); err != nil {
		return err
	}

	return nil
}

func NewHome(i uint) node.Home {
	pk, _ := keypair.NewStellarPrivateKey()

	h, _ := node.NewAddress([]byte{uint8(i)})
	return node.NewHome(h, pk)
}
