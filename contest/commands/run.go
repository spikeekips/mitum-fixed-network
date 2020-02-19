package commands

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/contest/common"
	"github.com/spikeekips/mitum/encoder"
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
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "keccak256")
	_ = hint.RegisterType(valuehash.SHA512{}.Hint().Type(), "keccak512")
	_ = hint.RegisterType((common.ContestAddress("")).Hint().Type(), "contest-node-address")
	_ = hint.RegisterType(isaac.INITBallotType, "init-ballot")
	_ = hint.RegisterType(isaac.ProposalBallotType, "proposal")
	_ = hint.RegisterType(isaac.SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(isaac.ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType(isaac.VoteproofType, "voteproof-genesis")
	_ = hint.RegisterType(isaac.BlockType, "block")
	_ = hint.RegisterType(isaac.BlockOperationType, "block-operation")
	_ = hint.RegisterType(isaac.BlockStatesType, "block-states")
	_ = hint.RegisterType(isaac.BlockStateType, "block-state")
}

func (cm RunCommand) generateInitialBlock() (isaac.BlockV0, error) {
	var initial isaac.BlockV0
	initial, err := common.NewContestBlock(
		isaac.Height(33),
		isaac.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	if err != nil {
		return isaac.BlockV0{}, err
	}

	return initial, nil
}

func (cm RunCommand) generateINITVoteproof(np *common.NodeProcess) (isaac.Voteproof, error) {
	round := isaac.Round(0)

	// empty Voteproof
	var ivp isaac.Voteproof
	for _, n := range np.AllNodes {
		ib, err := isaac.NewINITBallotV0FromLocalstate(n, round, nil)
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
		return nil, xerrors.Errorf("failed to make INIT Voteproof")
	}

	return ivp, nil
}

func (cm RunCommand) generateACCEPTVoteproof(
	np *common.NodeProcess, proposal valuehash.Hash,
) (isaac.BlockV0, isaac.Voteproof, error) {
	ivp := np.Localstate.LastINITVoteproof()

	newBlock, err := common.NewContestBlock(
		ivp.Height(),
		ivp.Round(),
		proposal,
		np.Localstate.LastBlock().Hash(),
	)
	if err != nil {
		return isaac.BlockV0{}, nil, err
	}

	var avp isaac.Voteproof
	for _, n := range np.AllNodes {
		ab, err := isaac.NewACCEPTBallotV0FromLocalstate(n, ivp.Round(), newBlock, nil)
		if err != nil {
			return isaac.BlockV0{}, nil, err
		}

		vp, err := np.Ballotbox.Vote(ab)
		if err != nil {
			return isaac.BlockV0{}, nil, err
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
		return isaac.BlockV0{}, nil, xerrors.Errorf("failed to make ACCEPT Voteproof")
	}

	return newBlock, avp, nil
}

func (cm RunCommand) generateBasement(nps []*common.NodeProcess) error {
	for _, np := range nps {
		localstate := np.Localstate
		lastBlock := localstate.LastBlock()

		ivpg := isaac.NewVoteproofGenesisV0(lastBlock.Height(), localstate.Policy().Threshold(), isaac.StageINIT)
		avpg := isaac.NewVoteproofGenesisV0(lastBlock.Height(), localstate.Policy().Threshold(), isaac.StageACCEPT)
		_ = localstate.SetLastINITVoteproof(ivpg)
		_ = localstate.SetLastACCEPTVoteproof(avpg)
	}

	ivps := map[isaac.Address]isaac.Voteproof{}
	for _, np := range nps {
		vp, err := cm.generateINITVoteproof(np)
		if err != nil {
			return err
		}
		ivps[np.Localstate.Node().Address()] = vp
	}

	for _, np := range nps {
		_ = np.Localstate.SetLastINITVoteproof(ivps[np.Localstate.Node().Address()])
	}

	proposal := valuehash.RandomSHA256()

	newBlocks := map[isaac.Address]isaac.Block{}
	avps := map[isaac.Address]isaac.Voteproof{}
	for _, np := range nps {
		newBlock, vp, err := cm.generateACCEPTVoteproof(np, proposal)
		if err != nil {
			return err
		}

		avps[np.Localstate.Node().Address()] = vp
		newBlocks[np.Localstate.Node().Address()] = newBlock
	}

	for _, np := range nps {
		_ = np.Localstate.SetLastACCEPTVoteproof(avps[np.Localstate.Node().Address()])

		newBlock := newBlocks[np.Localstate.Node().Address()]
		newBlock = newBlock.SetINITVoteproof(ivps[np.Localstate.Node().Address()])
		newBlock = newBlock.SetACCEPTVoteproof(avps[np.Localstate.Node().Address()])

		ob, err := np.Localstate.Storage().OpenBlockStorage(newBlock)
		if err != nil {
			return err
		} else if err := ob.Commit(); err != nil {
			return err
		}

		_ = np.Localstate.SetLastBlock(newBlock)
		np.Log().Debug().Interface("last_block", np.Localstate.LastBlock()).Msg("will start from here")
	}

	return nil
}

func (cm RunCommand) createNodeProcess(
	localstate *isaac.Localstate,
	log *zerolog.Logger,
) (*common.NodeProcess, error) {
	np, err := common.NewNodeProcess(localstate)
	if err != nil {
		return nil, err
	}

	_ = np.SetLogger(log.With().
		Str("node", np.Localstate.Node().Address().String()).
		Logger())

	{
		b, err := util.JSONMarshal(np.Localstate)
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

	var ns []*isaac.Localstate
	for i := 0; i < int(cm.Nodes); i++ {
		if nl, err := common.NewNode(i, initialBlock); err != nil {
			return err
		} else {
			ns = append(ns, nl)
		}
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

		threshold, err := isaac.NewThreshold(uint(nl.Nodes().Len()+1), 67)
		if err != nil {
			return err
		}
		_ = nl.Policy().SetThreshold(threshold)
	}

	nps := make([]*common.NodeProcess, len(ns))
	for i, nl := range ns {
		np, err := cm.createNodeProcess(nl, log)
		if err != nil {
			return err
		}
		nps[i] = np
	}

	for _, np := range nps {
		var nodes []*isaac.Localstate
		for _, other := range nps {
			nodes = append(nodes, other.Localstate)
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
