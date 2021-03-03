package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NewINITBallotV0Round0(local *Local) (ballot.INITBallotV0, error) {
	var m block.Manifest
	switch l, found, err := local.Storage().LastManifest(); {
	case !found:
		return ballot.INITBallotV0{}, xerrors.Errorf("last block not found")
	case err != nil:
		return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last block: %w", err)
	default:
		m = l
	}

	var avp base.Voteproof
	switch vp, found, err := local.BlockFS().LastVoteproof(base.StageACCEPT); {
	case !found:
		if m.Height() != base.PreGenesisHeight {
			return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last voteproof: %w", err)
		}
	case err != nil:
		return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last voteproof: %w", err)
	default:
		avp = vp
	}

	if avp != nil {
		return NewINITBallotV0WithVoteproof(local, local.Node().Address(), avp)
	}

	return ballot.NewINITBallotV0(
		local.Node().Address(),
		m.Height()+1,
		base.Round(0),
		m.Hash(),
		avp,
		avp,
	), nil
}

func NewINITBallotV0WithVoteproof(local *Local, node base.Address, voteproof base.Voteproof) (
	ballot.INITBallotV0, error,
) {
	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash
	var avp base.Voteproof
	switch voteproof.Stage() {
	case base.StageINIT:
		height = voteproof.Height()
		round = voteproof.Round() + 1
		switch t := voteproof.Majority().(type) {
		case ballot.INITBallotFact:
			previousBlock = t.PreviousBlock()
		case ballot.ACCEPTBallotFact:
			previousBlock = t.NewBlock()
		}

		switch vp, found, err := local.BlockFS().LastVoteproof(base.StageACCEPT); {
		case err != nil:
			return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last voteproof: %w", err)
		case found:
			avp = vp
		}
	case base.StageACCEPT:
		avp = voteproof
		height = voteproof.Height() + 1
		round = base.Round(0)
		if f, ok := voteproof.Majority().(ballot.ACCEPTBallotFact); !ok {
			return ballot.INITBallotV0{},
				xerrors.Errorf("invalid voteproof found; should have ACCEPTBallotFact, not %T", voteproof.Majority())
		} else {
			previousBlock = f.NewBlock()
		}
	}

	return ballot.NewINITBallotV0(
		node,
		height,
		round,
		previousBlock,
		voteproof,
		avp,
	), nil
}

func NewProposalV0(
	st storage.Storage,
	node base.Address,
	round base.Round,
	seals []valuehash.Hash,
	voteproof base.Voteproof,
) (
	ballot.ProposalV0, error,
) {
	var manifest block.Manifest
	switch l, found, err := st.LastManifest(); {
	case !found:
		return ballot.ProposalV0{}, storage.NotFoundError.Errorf("last manifest not found for NewProposalV0")
	case err != nil:
		return ballot.ProposalV0{}, err
	default:
		manifest = l
	}

	return ballot.NewProposalV0(
		node,
		manifest.Height()+1,
		round,
		seals,
		voteproof,
	), nil
}

func NewSIGNBallotV0(node base.Address, newBlock block.Block) ballot.SIGNBallotV0 {
	return ballot.NewSIGNBallotV0(
		node,
		newBlock.Height(),
		newBlock.Round(),
		newBlock.Proposal(),
		newBlock.Hash(),
	)
}

func NewACCEPTBallotV0(node base.Address, newBlock block.Block, voteproof base.Voteproof) ballot.ACCEPTBallotV0 {
	return ballot.NewACCEPTBallotV0(
		node,
		newBlock.Height(),
		newBlock.Round(),
		newBlock.Proposal(),
		newBlock.Hash(),
		voteproof,
	)
}

func SignSeal(b seal.Signer, local *Local) error {
	return b.Sign(local.Node().Privatekey(), local.Policy().NetworkID())
}
