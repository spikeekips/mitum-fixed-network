package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
)

type GenesisBlockV0Generator struct {
	localstate *Localstate
	ballotbox  *Ballotbox
	ops        []operation.Operation
}

func NewGenesisBlockV0Generator(localstate *Localstate, ops []operation.Operation) (*GenesisBlockV0Generator, error) {
	threshold, _ := base.NewThreshold(1, 100)

	return &GenesisBlockV0Generator{
		localstate: localstate,
		ballotbox: NewBallotbox(func() base.Threshold {
			return threshold
		}),
		ops: ops,
	}, nil
}

func (gg *GenesisBlockV0Generator) Generate() (block.Block, error) {
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

	var proposal ballot.Proposal
	if pr, err := gg.generateProposal(seals); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	initVoteproof := gg.localstate.LastINITVoteproof()

	var blk block.Block

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

			blk = bs.Block()
		}
	}

	return blk, nil
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

	blk, err := block.NewBlockV0(base.Height(-1), base.Round(0), genesisHash, genesisHash, nil, nil)
	if err != nil {
		return err
	}

	if bs, err := gg.localstate.Storage().OpenBlockStorage(blk); err != nil {
		return err
	} else if err := bs.Commit(); err != nil {
		return err
	}

	_ = gg.localstate.SetLastBlock(blk)

	return nil
}

func (gg *GenesisBlockV0Generator) generateProposal(seals []operation.Seal) (ballot.Proposal, error) {
	var operations []valuehash.Hash
	sealHashes := make([]valuehash.Hash, len(seals))
	for i := range seals {
		sl := seals[i]
		sealHashes[i] = sl.Hash()
		for _, op := range sl.Operations() {
			operations = append(operations, op.Hash())
		}
	}

	var proposal ballot.Proposal
	if pr, err := NewProposal(
		gg.localstate,
		base.Height(0),
		base.Round(0),
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
		base.Height(0),
		base.Round(0),
		previousBlock.Hash(),
		base.Round(0),
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

func (gg *GenesisBlockV0Generator) generateACCEPTVoteproof(newBlock block.Block) error {
	initVoteproof := gg.localstate.LastINITVoteproof()

	if ab, err := NewACCEPTBallotV0(
		gg.localstate,
		base.Height(0),
		base.Round(0),
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
