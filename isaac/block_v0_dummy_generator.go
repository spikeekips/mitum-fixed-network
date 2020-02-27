package isaac

import "github.com/spikeekips/mitum/seal"

type DummyBlocksV0Generator struct {
	genesisNode *Localstate
	localstates []*Localstate
	lastHeight  Height
	suffrage    Suffrage
	b           []byte
	allNodes    map[Address]*Localstate
	ballotboxes map[Address]*Ballotbox
	pms         map[Address]ProposalProcessor
}

func NewDummyBlocksV0Generator(
	genesisNode *Localstate, lastHeight Height, b []byte, suffrage Suffrage, localstates []*Localstate,
) (*DummyBlocksV0Generator, error) {
	allNodes := map[Address]*Localstate{}
	ballotboxes := map[Address]*Ballotbox{}
	pms := map[Address]ProposalProcessor{}

	threshold, _ := NewThreshold(uint(len(localstates)), 67)
	for _, l := range localstates {
		allNodes[l.Node().Address()] = l
		ballotboxes[l.Node().Address()] = NewBallotbox(func() Threshold {
			return threshold
		})
		pms[l.Node().Address()] = NewProposalProcessorV0(l)
	}

	return &DummyBlocksV0Generator{
		genesisNode: genesisNode,
		localstates: localstates,
		lastHeight:  lastHeight,
		suffrage:    suffrage,
		b:           b,
		allNodes:    allNodes,
		ballotboxes: ballotboxes,
		pms:         pms,
	}, nil
}

func (bg *DummyBlocksV0Generator) Generate() error {
	genesis, err := NewGenesisBlockV0Generator(bg.genesisNode, bg.b)
	if err != nil {
		return err
	} else if block, err := genesis.Generate(); err != nil {
		return err
	} else {
		for _, l := range bg.allNodes {
			if err := l.SetLastBlock(block); err != nil {
				return err
			}
		}
	}

	if err := bg.syncBlocks(bg.genesisNode); err != nil {
		return err
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
	var blocks []Block
	height := Height(0)
	for {
		if block, err := from.Storage().BlockByHeight(height); err != nil {
			break
		} else {
			blocks = append(blocks, block)
		}

		height++
	}

	for _, block := range blocks {
		for _, l := range bg.allNodes {
			if l.Node().Address().Equal(from.Node().Address()) {
				continue
			}

			if bs, err := l.Storage().OpenBlockStorage(block); err != nil {
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
		func(sl seal.Seal) (bool, error) {
			seals = append(seals, sl)
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

		for _, sl := range seals {
			if err := l.Storage().NewSeal(sl); err != nil {
				return err
			}
		}
	}

	var proposals []Proposal
	if err := from.Storage().Proposals(
		func(proposal Proposal) (bool, error) {
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
	var voteproofs []Voteproof
	if err := from.Storage().Voteproofs(
		func(voteproof Voteproof) (bool, error) {
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
			if voteproof.Stage() == StageINIT {
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
	round := Round(0)

	if err := bg.createINITVoteproof(round); err != nil {
		return err
	}

	var proposal Proposal
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
		proposal := acceptVoteproof.Majority().(ACCEPTBallotFact).Proposal()

		pm := bg.pms[l.Node().Address()]
		if bs, err := pm.ProcessACCEPT(proposal, acceptVoteproof, bg.b); err != nil {
			return err
		} else if err := bs.Commit(); err != nil {
			return err
		} else if err := bs.Block().IsValid(bg.b); err != nil {
			return err
		} else if err := l.SetLastBlock(bs.Block()); err != nil {
			return err
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITVoteproof(round Round) error {
	var ballots []INITBallot
	for _, l := range bg.allNodes {
		if ib, err := bg.createINITBallot(l, round); err != nil {
			return err
		} else {
			ballots = append(ballots, ib)
		}
	}

	for _, l := range bg.allNodes {
		for _, ballot := range ballots {
			if err := l.Storage().NewSeal(ballot); err != nil {
				return err
			}

			if vp, err := bg.ballotboxes[l.Node().Address()].Vote(ballot); err != nil {
				return err
			} else if vp.IsFinished() && !vp.IsClosed() {
				_ = l.SetLastINITVoteproof(vp)
			}
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITBallot(localstate *Localstate, round Round) (INITBallot, error) {
	previousBlock := localstate.LastBlock()

	var initBallot INITBallot
	if ib, err := NewINITBallotV0(
		localstate,
		previousBlock.Height()+1,
		round,
		previousBlock.Hash(),
		previousBlock.Round(),
		previousBlock.ACCEPTVoteproof(),
		bg.b,
	); err != nil {
		return nil, err
	} else {
		initBallot = ib
	}

	if err := localstate.Storage().NewSeal(initBallot); err != nil {
		return nil, err
	}

	return initBallot, nil
}

func (bg *DummyBlocksV0Generator) createProposal() (Proposal, error) {
	initVoteproof := bg.genesisNode.LastINITVoteproof()

	acting := bg.suffrage.Acting(initVoteproof.Height(), initVoteproof.Round())
	proposer := bg.allNodes[acting.Proposer().Address()]

	pr, err := NewProposal(proposer, initVoteproof.Height(), initVoteproof.Round(), nil, bg.b)
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

func (bg *DummyBlocksV0Generator) createACCEPTVoteproof(proposal Proposal) error {
	var ballots []ACCEPTBallot
	for _, l := range bg.allNodes {
		var newBlock Block

		initVoteproof := l.LastINITVoteproof()
		if b, err := bg.pms[l.Node().Address()].ProcessINIT(proposal.Hash(), initVoteproof, bg.b); err != nil {
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
			bg.b,
		); err != nil {
			return err
		} else {
			ballots = append(ballots, ab)
		}
	}

	for _, l := range bg.allNodes {
		for _, ballot := range ballots {
			if err := l.Storage().NewSeal(ballot); err != nil {
				return err
			}

			if vp, err := bg.ballotboxes[l.Node().Address()].Vote(ballot); err != nil {
				return err
			} else if vp.IsFinished() && !vp.IsClosed() {
				_ = l.SetLastACCEPTVoteproof(vp)
			}
		}
	}

	return nil
}