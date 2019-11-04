package main

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/configs"
	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/keypair"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type Node struct {
	*common.Logger
	homeState *isaac.HomeState
	nt        *contest_module.ChannelNetwork
	sc        *isaac.StateController
}

func NewNode(
	home node.Home,
	nodes []node.Node,
	globalConfig *configs.Config,
	config *configs.NodeConfig,
) (*Node, error) {
	rootLog := log.With().Str("node", home.Alias()).Logger()
	log_ := rootLog.With().Str("module", "node").Logger()

	lastBlock := config.Blocks[len(config.Blocks)-1].ToBlock()
	previousBlock := config.Blocks[len(config.Blocks)-2].ToBlock()

	homeState := isaac.NewHomeState(home, previousBlock).SetBlock(lastBlock)

	suffrage := config.Modules.Suffrage.(contest_module.SuffrageConfig).New(homeState, nodes, rootLog)
	ballotChecker := isaac.NewCompilerBallotChecker(homeState, suffrage)
	ballotChecker.SetLogger(rootLog)

	thr, _ := isaac.NewThreshold(suffrage.NumberOfActing(), *config.Policy.Threshold)
	if err := thr.Set(isaac.StageINIT, globalConfig.NumberOfNodes, *config.Policy.Threshold); err != nil {
		return nil, err
	}

	cm := isaac.NewCompiler(homeState, isaac.NewBallotbox(thr), ballotChecker)
	cm.SetLogger(rootLog)

	nt := contest_module.NewChannelNetwork(
		home,
		func(sl seal.Seal) (seal.Seal, error) {
			return sl, xerrors.Errorf("echo back")
		},
	)
	nt.SetLogger(rootLog)

	pv := config.Modules.ProposalValidator.(contest_module.ProposalValidatorConfig).New(homeState, rootLog)

	ballotMaker := config.Modules.BallotMaker.(contest_module.BallotMakerConfig).New(homeState, rootLog)

	var sc *isaac.StateController
	{ // state handlers
		bs := isaac.NewBootingStateHandler(homeState)
		bs.SetLogger(rootLog)

		js, err := isaac.NewJoinStateHandler(
			homeState,
			cm,
			nt,
			suffrage,
			ballotMaker,
			pv,
			*config.Policy.IntervalBroadcastINITBallotInJoin,
			*config.Policy.TimeoutWaitVoteResultInJoin,
		)
		if err != nil {
			return nil, err
		}
		js.SetLogger(rootLog)

		dp := config.Modules.ProposalMaker.(contest_module.ProposalMakerConfig).New(homeState, rootLog)

		cs, err := isaac.NewConsensusStateHandler(
			homeState,
			cm,
			nt,
			suffrage,
			ballotMaker,
			pv,
			dp,
			*config.Policy.TimeoutWaitBallot,
			*config.Policy.TimeoutWaitINITBallot,
		)
		if err != nil {
			return nil, err
		}
		cs.SetLogger(rootLog)

		ss := isaac.NewStoppedStateHandler()
		ss.SetLogger(rootLog)

		ssr := newSealStorage(config, rootLog)

		sc = isaac.NewStateController(homeState, cm, ssr, bs, js, cs, ss)
		sc.SetLogger(rootLog)
	}

	log_.Info().
		Object("config", config).
		Object("home", home).
		Object("homeState", homeState).
		Object("threshold", thr).
		Interface("suffrage", suffrage).
		Uint("number_of_acting", suffrage.NumberOfActing()).
		Msg("node created")

	n := &Node{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("node", home.Alias())
		}),
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
	started := time.Now()

	if err := no.nt.Start(); err != nil {
		return err
	}

	if err := no.sc.Start(); err != nil {
		return err
	}

	go func() {
		for m := range no.nt.Reader() {
			go func(m interface{}) {
				st := time.Now()
				err := no.sc.Receive(m)
				no.sc.Log().Debug().
					Err(err).
					Dur("elapsed", time.Since(st)).
					Msg("message received")
			}(m)
		}
	}()
	no.Log().Debug().Dur("elapsed", time.Since(started)).Msg("node started")

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

func newSealStorage(_ *configs.NodeConfig, l zerolog.Logger) isaac.SealStorage {
	ss := contest_module.NewMemorySealStorage()
	ss.SetLogger(l)

	return ss
}

func getAllNodesFromConfig(config *configs.Config) []node.Node {
	var nodeNames []string
	for n := range config.Nodes {
		nodeNames = append(nodeNames, n)
	}
	sort.Slice(
		nodeNames,
		func(i, j int) bool {
			var ni, nj int
			_, _ = fmt.Sscanf(nodeNames[i], "n%d", &ni)
			_, _ = fmt.Sscanf(nodeNames[j], "n%d", &nj)
			return ni < nj
		},
	)

	var nodeList []node.Node
	for i, name := range nodeNames[:config.NumberOfNodes] {
		n := NewHome(uint(i)).SetAlias(name)
		nodeList = append(nodeList, n)
	}

	return nodeList
}
