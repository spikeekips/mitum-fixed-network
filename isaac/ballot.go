package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
)

func NewINITBallotV0Round0(st storage.Storage, node base.Address) (ballot.INITBallotV0, error) {
	var m block.Manifest
	switch l, found, err := st.LastManifest(); {
	case !found:
		return ballot.INITBallotV0{}, xerrors.Errorf("last block not found")
	case err != nil:
		return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last block: %w", err)
	default:
		m = l
	}

	var avp base.Voteproof
	switch vp, found, err := st.LastVoteproof(base.StageACCEPT); {
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
		return NewINITBallotV0WithVoteproof(st, node, avp)
	}

	return ballot.NewINITBallotV0(
		node,
		m.Height()+1,
		base.Round(0),
		m.Hash(),
		avp,
	), nil
}

func NewINITBallotV0WithVoteproof(st storage.Storage, node base.Address, voteproof base.Voteproof) (
	ballot.INITBallotV0, error,
) {
	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash
	switch voteproof.Stage() {
	case base.StageINIT:
		height = voteproof.Height()
		round = voteproof.Round() + 1

		var manifest block.Manifest
		switch l, found, err := st.LastManifest(); {
		case !found:
			return ballot.INITBallotV0{}, xerrors.Errorf("last block not found: %w", err)
		case err != nil:
			return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last block: %w", err)
		default:
			manifest = l
		}
		if manifest.Height() != voteproof.Height()-1 {
			return ballot.INITBallotV0{},
				xerrors.Errorf("invalid init voteproof.Height(), %d; it should be lastBlock, %d + 1",
					voteproof.Height(), manifest.Height())
		}

		previousBlock = manifest.Hash()
	case base.StageACCEPT:
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
	), nil
}

func NewINITBallotV0(node base.Address, voteproof base.Voteproof) (ballot.INITBallotV0, error) {
	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash
	switch voteproof.Stage() {
	case base.StageACCEPT:
		if fact, ok := voteproof.Majority().(ballot.ACCEPTBallotFact); !ok {
			return ballot.INITBallotV0{}, xerrors.Errorf(
				"invalid accept voteproof found; majority fact is not accept ballot fact")
		} else {
			previousBlock = fact.NewBlock()
		}

		height = voteproof.Height() + 1
		round = base.Round(0)
	case base.StageINIT:
		if fact, ok := voteproof.Majority().(ballot.INITBallotFact); !ok {
			return ballot.INITBallotV0{}, xerrors.Errorf(
				"invalid init voteproof found; majority fact is not init ballot fact")
		} else {
			previousBlock = fact.PreviousBlock()
		}

		height = voteproof.Height()
		round = voteproof.Round() + 1
	default:
		return ballot.INITBallotV0{}, xerrors.Errorf("invalid voteproof stage, %v found", voteproof.Stage())
	}

	return ballot.NewINITBallotV0(
		node,
		height,
		round,
		previousBlock,
		voteproof,
	), nil
}

func NewProposalV0(st storage.Storage, node base.Address, round base.Round, operations, seals []valuehash.Hash) (
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
		operations,
		seals,
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

func SignSeal(b seal.Signer, localstate *Localstate) error {
	return b.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID())
}
