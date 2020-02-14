package commands

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/contest/common"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type RunCommand struct {
	Nodes uint `args:"" default:"${nodes}" help:"number of suffrage nodes"`
}

func (cm RunCommand) registerTypes() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "keccak256")
	_ = hint.RegisterType(valuehash.SHA512{}.Hint().Type(), "keccak512")
	_ = hint.RegisterType((common.ContestAddress("")).Hint().Type(), "contest-node-address")
	_ = hint.RegisterType(isaac.INITBallotType, "init-ballot")
	_ = hint.RegisterType(isaac.ProposalBallotType, "proposal")
	_ = hint.RegisterType(isaac.SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(isaac.ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType(isaac.VoteProofType, "voteproof-genesis")
}

func (cm RunCommand) generateInitialBlock() (isaac.Block, error) {
	var initial isaac.Block
	initial, err := common.NewContestBlock(
		isaac.Height(33),
		isaac.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	if err != nil {
		return nil, err
	}

	return initial, nil
}

func (cm RunCommand) generateINITVoteProof(np *common.NodeProcess) (isaac.VoteProof, error) {
	round := isaac.Round(0)

	// empty VoteProof
	var ivp isaac.VoteProof
	for _, n := range np.AllNodes {
		ib, err := isaac.NewINITBallotV0FromLocalState(n, round, nil)
		if err != nil {
			return nil, err
		}

		vp, err := np.Ballotbox.Vote(ib)
		if err != nil {
			return nil, err
		}

		if !vp.IsFinished() {
			continue
		} else if vp.IsClosed() {
			break
		}

		ivp = vp
		break
	}

	if ivp == nil {
		return nil, xerrors.Errorf("failed to make INIT VoteProof")
	}

	return ivp, nil
}

func (cm RunCommand) generateACCEPTVoteProof(
	np *common.NodeProcess, proposal valuehash.Hash,
) (isaac.Block, isaac.VoteProof, error) {
	ivp := np.LocalState.LastINITVoteProof()

	newBlock, err := common.NewContestBlock(
		ivp.Height(),
		ivp.Round(),
		proposal,
		np.LocalState.LastBlock().Hash(),
	)
	if err != nil {
		return nil, nil, err
	}

	var avp isaac.VoteProof
	for _, n := range np.AllNodes {
		ab, err := isaac.NewACCEPTBallotV0FromLocalState(n, ivp.Round(), newBlock, nil)
		if err != nil {
			return nil, nil, err
		}

		vp, err := np.Ballotbox.Vote(ab)
		if err != nil {
			return nil, nil, err
		}

		if !vp.IsFinished() {
			continue
		} else if vp.IsClosed() {
			break
		}

		avp = vp
		break
	}

	if avp == nil {
		return nil, nil, xerrors.Errorf("failed to make ACCEPT VoteProof")
	}

	return newBlock, avp, nil
}

func (cm RunCommand) generateBasement(nps []*common.NodeProcess) error {
	for _, np := range nps {
		localState := np.LocalState
		lastBlock := localState.LastBlock()

		ivpg := isaac.NewVoteProofGenesisV0(lastBlock.Height(), localState.Policy().Threshold(), isaac.StageINIT)
		avpg := isaac.NewVoteProofGenesisV0(lastBlock.Height(), localState.Policy().Threshold(), isaac.StageACCEPT)
		_ = localState.SetLastINITVoteProof(ivpg)
		_ = localState.SetLastACCEPTVoteProof(avpg)
	}

	ivps := map[isaac.Address]isaac.VoteProof{}
	for _, np := range nps {
		vp, err := cm.generateINITVoteProof(np)
		if err != nil {
			return err
		}
		ivps[np.LocalState.Node().Address()] = vp
	}

	for _, np := range nps {
		np.LocalState.SetLastINITVoteProof(ivps[np.LocalState.Node().Address()])
	}

	proposal := valuehash.RandomSHA256()

	newBlocks := map[isaac.Address]isaac.Block{}
	avps := map[isaac.Address]isaac.VoteProof{}
	for _, np := range nps {
		newBlock, vp, err := cm.generateACCEPTVoteProof(np, proposal)
		if err != nil {
			return err
		}

		avps[np.LocalState.Node().Address()] = vp
		newBlocks[np.LocalState.Node().Address()] = newBlock
	}

	for _, np := range nps {
		_ = np.LocalState.SetLastACCEPTVoteProof(avps[np.LocalState.Node().Address()])
		_ = np.LocalState.SetLastBlock(newBlocks[np.LocalState.Node().Address()])

		np.Log().Debug().Interface("last_block", np.LocalState.LastBlock()).Msg("will start from here")
	}

	return nil
}

func (cm RunCommand) createNodeProcess(
	localState *isaac.LocalState,
	initialBlock isaac.Block,
	log *zerolog.Logger,
) (*common.NodeProcess, error) {
	np, err := common.NewNodeProcess(localState, initialBlock)
	if err != nil {
		return nil, err
	}

	_ = np.SetLogger(log.With().
		Str("node", np.LocalState.Node().Address().String()).
		Logger())

	{
		b, err := util.JSONMarshal(np.LocalState)
		if err != nil {
			return nil, err
		}
		np.Log().Debug().RawJSON("local_states", b).Msg("node process created")
	}

	return np, nil
}

func (cm RunCommand) startNodes(nodeProcesses []*common.NodeProcess, exitHooks *[]func()) error {
	var wg sync.WaitGroup
	wg.Add(len(nodeProcesses))

	errChan := make(chan error)
	for _, np := range nodeProcesses {
		np := np

		*exitHooks = append(*exitHooks, func() {
			_ = np.Stop()
		})

		go func(np *common.NodeProcess) {
			errChan <- np.Start()
			wg.Done()
		}(np)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err == nil {
			continue
		}

		log.Error().Err(err).Msg("failed to start NodeProcess")

		return err
	}

	return nil
}

func (cm RunCommand) Run(_ *CommonFlags, log *zerolog.Logger, exitHooks *[]func()) error {
	cm.registerTypes()

	initialBlock, err := cm.generateInitialBlock()
	if err != nil {
		return err
	}

	var ns []*isaac.LocalState
	for i := 0; i < int(cm.Nodes); i++ {
		nl := common.NewNode(i)
		ns = append(ns, nl)
	}

	for _, nl := range ns {
		for _, other := range ns {
			if nl.Node().Address().Equal(other.Node().Address()) {
				continue
			}
			if err := nl.Nodes().Add(other.Node()); err != nil {
				return err
			}
		}

		// set threshold
		threshold, err := isaac.NewThreshold(uint(nl.Nodes().Len()+1), 67)
		if err != nil {
			return err
		}
		_ = nl.Policy().SetThreshold(threshold)
	}

	nps := make([]*common.NodeProcess, len(ns))
	for _, nl := range ns {
		np, err := cm.createNodeProcess(nl, initialBlock, log)
		if err != nil {
			return err
		}

		nps = append(nps, np)
	}

	for _, np := range nps {
		var nodes []*isaac.LocalState
		for _, other := range nps {
			nodes = append(nodes, other.LocalState)
		}

		np.AllNodes = nodes
	}

	if err := cm.generateBasement(nps); err != nil {
		return err
	}

	if err := cm.startNodes(nps, exitHooks); err != nil {
		return err
	}

	return common.LongRunningCommandError
}
