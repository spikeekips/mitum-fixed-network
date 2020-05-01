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
	allNodes := map[base.Address]*Localstate{}
	ballotboxes := map[base.Address]*Ballotbox{}
	pms := map[base.Address]ProposalProcessor{}

	threshold, _ := base.NewThreshold(uint(len(localstates)), 67)
	for _, l := range localstates {
		allNodes[l.Node().Address()] = l
		ballotboxes[l.Node().Address()] = NewBallotbox(func() base.Threshold {
			return threshold
		})
		pms[l.Node().Address()] = NewProposalProcessorV0(l)
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
		lastBlock, err := bg.genesisNode.Storage().LastBlock()
		if err != nil {
			return err
		} else if err := lastBlock.Manifest().IsValid(bg.genesisNode.Policy().NetworkID()); err != nil {
			return err
		}
		lastHeight = lastBlock.Height()

		if lastHeight >= bg.lastHeight {
			return nil
		}
	}

	if lastHeight == base.NilHeight {
		genesis, err := NewGenesisBlockV0Generator(bg.genesisNode, nil)
		if err != nil {
			return err
		} else if blk, err := genesis.Generate(); err != nil {
			return err
		} else {
			for _, l := range bg.allNodes {
				if err := l.SetLastBlock(blk); err != nil {
					return err
				}
			}
		}

		if err := bg.syncBlocks(bg.genesisNode); err != nil {
			return err
		}
	}

	for {
		if err := bg.createNextBlock(); err != nil {
			return err
		}

		if bg.genesisNode.LastBlock().Height() == bg.lastHeight {
			break
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) syncBlocks(from *Localstate) error {
	var blocks []block.Block
	height := base.Height(0)
	for {
		if blk, err := from.Storage().BlockByHeight(height); err != nil {
			if xerrors.Is(err, storage.NotFoundError) {
				break
			}
			return err
		} else {
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

	if err := bg.syncVoteproofs(from); err != nil {
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

func (bg *DummyBlocksV0Generator) syncVoteproofs(from *Localstate) error {
	var voteproofs []base.Voteproof
	if err := from.Storage().Voteproofs(
		func(voteproof base.Voteproof) (bool, error) {
			voteproofs = append(voteproofs, voteproof)
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

		for _, voteproof := range voteproofs {
			if voteproof.Stage() == base.StageINIT {
				if err := l.Storage().NewINITVoteproof(voteproof); err != nil {
					return err
				}
			} else {
				if err := l.Storage().NewACCEPTVoteproof(voteproof); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createNextBlock() error {
	round := base.Round(0)

	if err := bg.createINITVoteproof(round); err != nil {
		return err
	}

	var proposal ballot.Proposal
	if pr, err := bg.createProposal(); err != nil {
		return err
	} else {
		proposal = pr
	}

	if err := bg.createACCEPTVoteproof(proposal); err != nil {
		return err
	}

	if err := bg.finish(); err != nil {
		return err
	}

	return nil
}

func (bg *DummyBlocksV0Generator) finish() error {
	for _, l := range bg.allNodes {
		acceptVoteproof := bg.genesisNode.LastACCEPTVoteproof()
		proposal := acceptVoteproof.Majority().(ballot.ACCEPTBallotFact).Proposal()

		pm := bg.pms[l.Node().Address()]
		if bs, err := pm.ProcessACCEPT(proposal, acceptVoteproof); err != nil {
			return err
		} else if err := bs.Block().IsValid(bg.networkID); err != nil {
			return err
		} else if err := bs.Commit(); err != nil {
			return err
		} else if err := l.SetLastBlock(bs.Block()); err != nil {
			return err
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITVoteproof(round base.Round) error {
	var ballots []ballot.INITBallot
	var seals []seal.Seal
	for _, l := range bg.allNodes {
		if ib, err := bg.createINITBallot(l, round); err != nil {
			return err
		} else {
			ballots = append(ballots, ib)
			seals = append(seals, ib)
		}
	}

	for _, l := range bg.allNodes {
		if err := l.Storage().NewSeals(seals); err != nil {
			return err
		}

		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				_ = l.SetLastINITVoteproof(voteproof)
			}
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITBallot(
	localstate *Localstate, round base.Round,
) (ballot.INITBallot, error) {
	previousBlock := localstate.LastBlock()

	var initBallot ballot.INITBallot
	if ib, err := NewINITBallotV0(
		localstate,
		previousBlock.Height()+1,
		round,
		previousBlock.Hash(),
		previousBlock.Round(),
		previousBlock.ACCEPTVoteproof(),
		bg.networkID,
	); err != nil {
		return nil, err
	} else {
		initBallot = ib
	}

	if err := localstate.Storage().NewSeals([]seal.Seal{initBallot}); err != nil {
		return nil, err
	}

	return initBallot, nil
}

func (bg *DummyBlocksV0Generator) createProposal() (ballot.Proposal, error) {
	initVoteproof := bg.genesisNode.LastINITVoteproof()

	acting := bg.suffrage.Acting(initVoteproof.Height(), initVoteproof.Round())
	proposer := bg.allNodes[acting.Proposer().Address()]

	pr, err := NewProposal(proposer, initVoteproof.Height(), initVoteproof.Round(), nil, nil, bg.networkID)
	if err != nil {
		return nil, err
	}

	for _, l := range bg.allNodes {
		if err := l.Storage().NewProposal(pr); err != nil {
			return nil, err
		}
	}

	return pr, nil
}

func (bg *DummyBlocksV0Generator) createACCEPTVoteproof(proposal ballot.Proposal) error {
	var ballots []ballot.ACCEPTBallot
	var seals []seal.Seal
	for _, l := range bg.allNodes {
		var newBlock block.Block

		initVoteproof := l.LastINITVoteproof()
		if b, err := bg.pms[l.Node().Address()].ProcessINIT(proposal.Hash(), initVoteproof); err != nil {
			return err
		} else if newBlock == nil {
			newBlock = b
		}

		if ab, err := NewACCEPTBallotV0(
			l,
			newBlock.Height(),
			newBlock.Round(),
			newBlock,
			initVoteproof,
			bg.networkID,
		); err != nil {
			return err
		} else {
			ballots = append(ballots, ab)
			seals = append(seals, ab)
		}
	}

	for _, l := range bg.allNodes {
		if err := l.Storage().NewSeals(seals); err != nil {
			return err
		}

		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				_ = l.SetLastACCEPTVoteproof(voteproof)
			}
		}
	}

	return nil
}
