package isaac

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NewINITBallotV0Round0(node base.Node, db storage.Database) (ballot.INITV0, error) {
	var m block.Manifest
	switch l, found, err := db.LastManifest(); {
	case err != nil:
		return ballot.INITV0{}, errors.Wrap(err, "failed to get last block")
	case !found:
		return ballot.INITV0{}, errors.Errorf("last block not found")
	default:
		m = l
	}

	avp := db.LastVoteproof(base.StageACCEPT)

	return ballot.NewINITV0(
		node.Address(),
		m.Height()+1,
		base.Round(0),
		m.Hash(),
		avp,
		avp,
	), nil
}

func NewINITBallotV0WithVoteproof(
	node base.Node,
	db storage.Database,
	voteproof base.Voteproof,
) (ballot.INITV0, error) {
	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash
	var avp base.Voteproof
	switch voteproof.Stage() {
	case base.StageINIT:
		height = voteproof.Height()
		round = voteproof.Round() + 1
		switch t := voteproof.Majority().(type) {
		case ballot.INITFact:
			previousBlock = t.PreviousBlock()
		case ballot.ACCEPTFact:
			previousBlock = t.NewBlock()
		}

		avp = db.LastVoteproof(base.StageACCEPT)
	case base.StageACCEPT:
		avp = voteproof
		height = voteproof.Height() + 1
		round = base.Round(0)
		f, ok := voteproof.Majority().(ballot.ACCEPTFact)
		if !ok {
			return ballot.INITV0{},
				errors.Errorf("invalid voteproof found; should have ACCEPTBallotFact, not %T", voteproof.Majority())
		}
		previousBlock = f.NewBlock()
	}

	return ballot.NewINITV0(
		node.Address(),
		height,
		round,
		previousBlock,
		voteproof,
		avp,
	), nil
}

func NewProposalV0(
	db storage.Database,
	node base.Address,
	round base.Round,
	seals []valuehash.Hash,
	voteproof base.Voteproof,
) (
	ballot.ProposalV0, error,
) {
	var manifest block.Manifest
	switch l, found, err := db.LastManifest(); {
	case err != nil:
		return ballot.ProposalV0{}, err
	case !found:
		return ballot.ProposalV0{}, util.NotFoundError.Errorf("last manifest not found for NewProposalV0")
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

func NewSIGNBallotV0(node base.Address, newBlock block.Block) ballot.SIGNV0 {
	return ballot.NewSIGNV0(
		node,
		newBlock.Height(),
		newBlock.Round(),
		newBlock.Proposal(),
		newBlock.Hash(),
	)
}

func NewACCEPTBallotV0(node base.Address, newBlock block.Block, voteproof base.Voteproof) ballot.ACCEPTV0 {
	return ballot.NewACCEPTV0(
		node,
		newBlock.Height(),
		newBlock.Round(),
		newBlock.Proposal(),
		newBlock.Hash(),
		voteproof,
	)
}
