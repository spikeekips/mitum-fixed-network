package isaac

import (
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type GenesisBlockV0Generator struct {
	localstate *Localstate
	b          []byte
	ballotbox  *Ballotbox
}

func NewGenesisBlockV0Generator(localstate *Localstate, b []byte) (*GenesisBlockV0Generator, error) {
	threshold, _ := NewThreshold(1, 100)

	return &GenesisBlockV0Generator{
		localstate: localstate,
		b:          b,
		ballotbox: NewBallotbox(func() Threshold {
			return threshold
		}),
	}, nil
}

func (gg *GenesisBlockV0Generator) Generate() (Block, error) {
	if err := gg.generatePreviousBlock(); err != nil {
		return nil, err
	}

	if err := gg.generateINITVoteproof(); err != nil {
		return nil, err
	}

	var proposal Proposal
	if pr, err := gg.generateProposal(); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	initVoteproof := gg.localstate.LastINITVoteproof()

	var block Block

	pm := NewProposalProcessorV0(gg.localstate)
	if bk, err := pm.ProcessINIT(proposal.Hash(), initVoteproof, gg.b); err != nil {
		return nil, err
	} else if err := gg.generateACCEPTVoteproof(bk); err != nil {
		return nil, err
	} else {
		acceptVoteproof := gg.localstate.LastACCEPTVoteproof()
		if bs, err := pm.ProcessACCEPT(proposal.Hash(), acceptVoteproof, gg.b); err != nil {
			return nil, err
		} else if err := bs.Commit(); err != nil {
			return nil, err
		} else {
			_ = gg.localstate.SetLastBlock(bs.Block())

			block = bs.Block()
		}
	}

	return block, nil
}

func (gg *GenesisBlockV0Generator) generatePreviousBlock() error {
	// NOTE the privatekey of local node is melted into genesis previous block;
	// it means, genesis block contains who creates it.
	var genesisHash valuehash.Hash
	if sig, err := gg.localstate.Node().Privatekey().Sign(gg.b); err != nil {
		return err
	} else {
		genesisHash = valuehash.NewDummy(sig)
	}

	block, err := NewBlockV0(
		Height(-1),
		Round(0),
		genesisHash,
		genesisHash,
		nil,
		nil,
		gg.b,
	)
	if err != nil {
		return err
	}

	if bs, err := gg.localstate.Storage().OpenBlockStorage(block); err != nil {
		return err
	} else if err := bs.Commit(); err != nil {
		return err
	}

	_ = gg.localstate.SetLastBlock(block)

	return nil
}

func (gg *GenesisBlockV0Generator) generateProposal() (Proposal, error) {
	var proposal Proposal
	if pr, err := NewProposal(gg.localstate, Height(0), Round(0), nil, gg.b); err != nil {
		return nil, err
	} else if err := gg.localstate.Storage().NewProposal(pr); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	return proposal, nil
}

func (gg *GenesisBlockV0Generator) generateINITVoteproof() error {
	previousBlock := gg.localstate.LastBlock()

	if ib, err := NewINITBallotV0(
		gg.localstate,
		Height(0),
		Round(0),
		previousBlock.Hash(),
		Round(0),
		nil,
		gg.b,
	); err != nil {
		return err
	} else if vp, err := gg.ballotbox.Vote(ib); err != nil {
		return err
	} else {
		if !vp.IsFinished() {
			return xerrors.Errorf("something wrong, INITVoteproof should be finished, but not")
		} else {
			if err := gg.localstate.Storage().NewSeal(ib); err != nil {
				return err
			} else if err := gg.localstate.Storage().NewINITVoteproof(vp); err != nil {
				return err
			}

			_ = gg.localstate.SetLastINITVoteproof(vp)
		}
	}

	return nil
}

func (gg *GenesisBlockV0Generator) generateACCEPTVoteproof(newBlock Block) error {
	initVoteproof := gg.localstate.LastINITVoteproof()

	if ab, err := NewACCEPTBallotV0(
		gg.localstate,
		Height(0),
		Round(0),
		newBlock,
		initVoteproof,
		gg.b,
	); err != nil {
		return err
	} else if err := gg.localstate.Storage().NewSeal(ab); err != nil {
		return err
	} else if vp, err := gg.ballotbox.Vote(ab); err != nil {
		return err
	} else {
		if !vp.IsFinished() {
			return xerrors.Errorf("something wrong, ACCEPTVoteproof should be finished, but not")
		} else {
			if err := gg.localstate.Storage().NewSeal(ab); err != nil {
				return err
			} else if err := gg.localstate.Storage().NewACCEPTVoteproof(vp); err != nil {
				return err
			}

			_ = gg.localstate.SetLastACCEPTVoteproof(vp)
		}
	}

	return nil
}
