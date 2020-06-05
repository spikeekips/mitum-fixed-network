package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"golang.org/x/xerrors"
)

type DummyBlocksV0Generator struct {
	genesisNode *Localstate
	localstates []*Localstate
	lastHeight  base.Height
	suffrage    base.Suffrage
	networkID   []byte
	allNodes    map[base.Address]*Localstate
	ballotboxes map[base.Address]*Ballotbox
	pms         map[base.Address]ProposalProcessor
}

func NewDummyBlocksV0Generator(
	genesisNode *Localstate, lastHeight base.Height, suffrage base.Suffrage, localstates []*Localstate,
) (*DummyBlocksV0Generator, error) {
	if lastHeight <= base.NilHeight {
		return nil, xerrors.Errorf("last height must not be nil height, %v", base.NilHeight)
	}

	allNodes := map[base.Address]*Localstate{}
	ballotboxes := map[base.Address]*Ballotbox{}
	pms := map[base.Address]ProposalProcessor{}

	threshold, _ := base.NewThreshold(uint(len(localstates)), 67)
	for _, l := range localstates {
		allNodes[l.Node().Address()] = l
		ballotboxes[l.Node().Address()] = NewBallotbox(func() base.Threshold {
			return threshold
		})
		pms[l.Node().Address()] = NewProposalProcessorV0(l, suffrage)
	}

	return &DummyBlocksV0Generator{
		genesisNode: genesisNode,
		localstates: localstates,
		lastHeight:  lastHeight,
		suffrage:    suffrage,
		networkID:   genesisNode.Policy().NetworkID(),
		allNodes:    allNodes,
		ballotboxes: ballotboxes,
		pms:         pms,
	}, nil
}

func (bg *DummyBlocksV0Generator) Generate(ignoreExists bool) error {
	lastHeight := base.NilHeight
	if !ignoreExists {
		if l, found, err := bg.genesisNode.Storage().LastManifest(); !found {
			return storage.NotFoundError.Errorf("last manifest not found")
		} else if err != nil {
			return err
		} else if err := l.IsValid(bg.genesisNode.Policy().NetworkID()); err != nil {
			return err
		} else {
			lastHeight = l.Height()
		}

		if lastHeight >= bg.lastHeight {
			return nil
		}
	}

	if lastHeight == base.NilHeight {
		genesis, err := NewGenesisBlockV0Generator(bg.genesisNode, nil)
		if err != nil {
			return err
		} else if _, err := genesis.Generate(); err != nil {
			return err
		}

		if err := bg.syncBlocks(bg.genesisNode); err != nil {
			return err
		}
	}

end:
	for {
		if err := bg.createNextBlock(); err != nil {
			return err
		}

		switch l, found, err := bg.genesisNode.Storage().LastManifest(); {
		case !found:
			return storage.NotFoundError.Errorf("last manifest not found")
		case err != nil:
			return err
		case l.Height() == bg.lastHeight:
			break end
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) syncBlocks(from *Localstate) error {
	var blocks []block.Block
	height := base.PreGenesisHeight

end:
	for {
		switch blk, found, err := from.Storage().BlockByHeight(height); {
		case err != nil:
			return err
		case !found:
			break end
		default:
			blocks = append(blocks, blk)
		}

		height++
	}

	if len(blocks) < 1 {
		return xerrors.Errorf("empty blocks for syncing blocks")
	}

	for _, blk := range blocks {
		for _, l := range bg.allNodes {
			if l.Node().Address().Equal(from.Node().Address()) {
				continue
			}

			if bs, err := l.Storage().OpenBlockStorage(blk); err != nil {
				return err
			} else if err := bs.SetBlock(blk); err != nil {
				return err
			} else if err := bs.Commit(); err != nil {
				return err
			}
		}
	}

	if err := bg.syncSeals(from); err != nil {
		return err
	}

	return nil
}

func (bg *DummyBlocksV0Generator) syncSeals(from *Localstate) error {
	var seals []seal.Seal
	if err := from.Storage().Seals(
		func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
			seals = append(seals, sl)
			return true, nil
		},
		true,
		true,
	); err != nil {
		return err
	}

	for _, l := range bg.allNodes {
		if l.Node().Address().Equal(from.Node().Address()) {
			continue
		}

		if err := l.Storage().NewSeals(seals); err != nil {
			return err
		}
	}

	var proposals []ballot.Proposal
	if err := from.Storage().Proposals(
		func(proposal ballot.Proposal) (bool, error) {
			proposals = append(proposals, proposal)
			return true, nil
		},
		true,
	); err != nil {
		return err
	}

	for _, l := range bg.allNodes {
		if l.Node().Address().Equal(from.Node().Address()) {
			continue
		}

		for _, proposal := range proposals {
			if err := l.Storage().NewProposal(proposal); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createNextBlock() error {
	var ivm map[base.Address]base.Voteproof
	if v, err := bg.createINITVoteproof(); err != nil {
		return err
	} else {
		ivm = v
	}

	var proposal ballot.Proposal
	if pr, err := bg.createProposal(ivm[bg.genesisNode.Node().Address()]); err != nil {
		return err
	} else {
		proposal = pr
	}

	var avm map[base.Address]base.Voteproof
	if v, err := bg.createACCEPTVoteproof(proposal, ivm); err != nil {
		return err
	} else {
		avm = v
	}

	if err := bg.finish(avm); err != nil {
		return err
	}

	return nil
}

func (bg *DummyBlocksV0Generator) finish(vm map[base.Address]base.Voteproof) error {
	for _, l := range bg.allNodes {
		var vp base.Voteproof
		if v, found := vm[l.Node().Address()]; !found {
			return xerrors.Errorf("failed to find voteproofs for all nodes")
		} else {
			vp = v
		}

		proposal := vp.Majority().(ballot.ACCEPTBallotFact).Proposal()

		pm := bg.pms[l.Node().Address()]
		if bs, err := pm.ProcessACCEPT(proposal, vp); err != nil {
			return err
		} else if err := bs.Block().IsValid(bg.networkID); err != nil {
			return err
		} else if err := bs.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITVoteproof() (map[base.Address]base.Voteproof, error) {
	var ballots []ballot.INITBallot
	var seals []seal.Seal
	for _, l := range bg.allNodes {
		if ib, err := bg.createINITBallot(l); err != nil {
			return nil, err
		} else {
			ballots = append(ballots, ib)
			seals = append(seals, ib)
		}
	}

	vm := map[base.Address]base.Voteproof{}
	for _, l := range bg.allNodes {
		if err := l.Storage().NewSeals(seals); err != nil {
			return nil, err
		}

		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return nil, err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				vm[l.Node().Address()] = voteproof
			}
		}
	}

	if len(vm) != len(bg.allNodes) {
		return nil, xerrors.Errorf("failed to create INIT Voteproof")
	}

	return vm, nil
}

func (bg *DummyBlocksV0Generator) createINITBallot(localstate *Localstate) (ballot.INITBallot, error) {
	var baseBallot ballot.INITBallotV0
	if b, err := NewINITBallotV0Round0(localstate.Storage(), localstate.Node().Address()); err != nil {
		return nil, err
	} else if err := SignSeal(&b, localstate); err != nil {
		return nil, err
	} else {
		baseBallot = b
	}

	if err := localstate.Storage().NewSeals([]seal.Seal{baseBallot}); err != nil {
		return nil, err
	}

	return baseBallot, nil
}

func (bg *DummyBlocksV0Generator) createProposal(voteproof base.Voteproof) (ballot.Proposal, error) {
	acting := bg.suffrage.Acting(voteproof.Height(), voteproof.Round())
	proposer := bg.allNodes[acting.Proposer()]

	pr := ballot.NewProposalV0(
		proposer.Node().Address(),
		voteproof.Height(),
		voteproof.Round(),
		nil,
		nil,
	)
	if err := SignSeal(&pr, proposer); err != nil {
		return nil, err
	}

	for _, l := range bg.allNodes {
		if err := l.Storage().NewProposal(pr); err != nil {
			return nil, err
		}
	}

	return pr, nil
}

func (bg *DummyBlocksV0Generator) createACCEPTVoteproof(proposal ballot.Proposal, ivm map[base.Address]base.Voteproof) (
	map[base.Address]base.Voteproof, error,
) {
	var ballots []ballot.ACCEPTBallot
	var seals []seal.Seal
	for _, l := range bg.allNodes {
		var newBlock block.Block

		ivp := ivm[l.Node().Address()]
		if b, err := bg.pms[l.Node().Address()].ProcessINIT(proposal.Hash(), ivp); err != nil {
			return nil, err
		} else if newBlock == nil {
			newBlock = b
		}

		ab := NewACCEPTBallotV0(l.Node().Address(), newBlock, ivp)
		if err := SignSeal(&ab, l); err != nil {
			return nil, err
		} else {
			ballots = append(ballots, ab)
			seals = append(seals, ab)
		}
	}

	vm := map[base.Address]base.Voteproof{}
	for _, l := range bg.allNodes {
		if err := l.Storage().NewSeals(seals); err != nil {
			return nil, err
		}

		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return nil, err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				vm[l.Node().Address()] = voteproof
			}
		}
	}

	if len(vm) != len(bg.allNodes) {
		return nil, xerrors.Errorf("failed to create voteproofs for all nodes")
	}

	return vm, nil
}
