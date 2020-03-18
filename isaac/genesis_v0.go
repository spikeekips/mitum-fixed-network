package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type GenesisBlockV0Generator struct {
	localstate *Localstate
	ballotbox  *Ballotbox
	ops        []operation.Operation
}

func NewGenesisBlockV0Generator(localstate *Localstate, ops []operation.Operation) (*GenesisBlockV0Generator, error) {
	threshold, _ := NewThreshold(1, 100)

	return &GenesisBlockV0Generator{
		localstate: localstate,
		ballotbox: NewBallotbox(func() Threshold {
			return threshold
		}),
		ops: ops,
	}, nil
}

func (gg *GenesisBlockV0Generator) Generate() (Block, error) {
	if err := gg.generatePreviousBlock(); err != nil {
		return nil, err
	}

	if err := gg.generateINITVoteproof(); err != nil {
		return nil, err
	}

	var seals []operation.Seal
	if sls, err := gg.generateOperationSeal(); err != nil {
		return nil, err
	} else {
		seals = sls
	}

	var proposal Proposal
	if pr, err := gg.generateProposal(seals); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	initVoteproof := gg.localstate.LastINITVoteproof()

	var block Block

	pm := NewProposalProcessorV0(gg.localstate)
	pm.SetLogger(log)

	if bk, err := pm.ProcessINIT(proposal.Hash(), initVoteproof); err != nil {
		return nil, err
	} else if err := gg.generateACCEPTVoteproof(bk); err != nil {
		return nil, err
	} else {
		acceptVoteproof := gg.localstate.LastACCEPTVoteproof()
		if bs, err := pm.ProcessACCEPT(proposal.Hash(), acceptVoteproof); err != nil {
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

func (gg *GenesisBlockV0Generator) generateOperationSeal() ([]operation.Seal, error) {
	if len(gg.ops) < 1 {
		return nil, nil
	}

	var seals []operation.Seal
	if sl, err := operation.NewSeal(
		gg.localstate.Node().Privatekey(),
		gg.ops,
		gg.localstate.Policy().NetworkID(),
	); err != nil {
		return nil, err
	} else if err := gg.localstate.Storage().NewSeals([]seal.Seal{sl}); err != nil {
		return nil, err
	} else {
		seals = append(seals, sl)
	}

	return seals, nil
}

func (gg *GenesisBlockV0Generator) generatePreviousBlock() error {
	// NOTE the privatekey of local node is melted into genesis previous block;
	// it means, genesis block contains who creates it.
	var genesisHash valuehash.Hash
	if sig, err := gg.localstate.Node().Privatekey().Sign(gg.localstate.Policy().NetworkID()); err != nil {
		return err
	} else {
		genesisHash = valuehash.NewDummy(sig)
	}

	block, err := NewBlockV0(Height(-1), Round(0), genesisHash, genesisHash, nil, nil)
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

func (gg *GenesisBlockV0Generator) generateProposal(seals []operation.Seal) (Proposal, error) {
	var operations []valuehash.Hash
	sealHashes := make([]valuehash.Hash, len(seals))
	for i := range seals {
		sl := seals[i]
		sealHashes[i] = sl.Hash()
		for _, op := range sl.Operations() {
			operations = append(operations, op.Hash())
		}
	}

	var proposal Proposal
	if pr, err := NewProposal(
		gg.localstate,
		Height(0),
		Round(0),
		operations,
		sealHashes,
		gg.localstate.Policy().NetworkID(),
	); err != nil {
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
		gg.localstate.Policy().NetworkID(),
	); err != nil {
		return err
	} else if voteproof, err := gg.ballotbox.Vote(ib); err != nil {
		return err
	} else {
		if !voteproof.IsFinished() {
			return xerrors.Errorf("something wrong, INITVoteproof should be finished, but not")
		} else {
			if err := gg.localstate.Storage().NewSeals([]seal.Seal{ib}); err != nil {
				return err
			} else if err := gg.localstate.Storage().NewINITVoteproof(voteproof); err != nil {
				return err
			}

			_ = gg.localstate.SetLastINITVoteproof(voteproof)
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
		gg.localstate.Policy().NetworkID(),
	); err != nil {
		return err
	} else if err := gg.localstate.Storage().NewSeals([]seal.Seal{ab}); err != nil {
		return err
	} else if voteproof, err := gg.ballotbox.Vote(ab); err != nil {
		return err
	} else {
		if !voteproof.IsFinished() {
			return xerrors.Errorf("something wrong, ACCEPTVoteproof should be finished, but not")
		} else {
			if err := gg.localstate.Storage().NewSeals([]seal.Seal{ab}); err != nil {
				return err
			} else if err := gg.localstate.Storage().NewACCEPTVoteproof(voteproof); err != nil {
				return err
			}

			_ = gg.localstate.SetLastACCEPTVoteproof(voteproof)
		}
	}

	return nil
}
