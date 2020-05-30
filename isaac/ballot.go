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
	switch l, err := st.LastManifest(); {
	case err != nil:
		return ballot.INITBallotV0{}, xerrors.Errorf("last block not found: %w", err)
	default:
		m = l
	}

	var avp base.Voteproof
	if vp, err := st.LastVoteproof(base.StageACCEPT); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return ballot.INITBallotV0{}, xerrors.Errorf("last voteproof not found: %w", err)
		} else if m.Height() != base.PreGenesisHeight {
			return ballot.INITBallotV0{}, xerrors.Errorf("failed to get last voteproof: %w", err)
		}
	} else {
		avp = vp
	}

	return ballot.NewINITBallotV0(
		node,
		m.Height()+1,
		base.Round(0),
		m.Hash(),
		avp,
	), nil
}

func NewINITBallotV0WithVoteproof(st storage.Storage, node base.Address, round base.Round, voteproof base.Voteproof) (
	ballot.INITBallotV0, error,
) {
	var manifest block.Manifest
	switch l, err := st.LastManifest(); {
	case err != nil:
		return ballot.INITBallotV0{}, xerrors.Errorf("last block not found: %w", err)
	default:
		manifest = l
	}

	// TODO height and round should rely on voteproof
	return ballot.NewINITBallotV0(
		node,
		manifest.Height()+1,
		round,
		manifest.Hash(),
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
	if l, err := st.LastManifest(); err != nil {
		return ballot.ProposalV0{}, err
	} else {
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
