package isaac

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NewINITBallotRound0(
	n base.Address,
	db storage.Database,
	pk key.Privatekey,
	networkID base.NetworkID,
) (base.INITBallot, error) {
	var m block.Manifest
	switch l, found, err := db.LastManifest(); {
	case err != nil:
		return nil, errors.Wrap(err, "failed to get last block")
	case !found:
		return nil, errors.Errorf("last block not found")
	default:
		m = l
	}

	avp := db.LastVoteproof(base.StageACCEPT)

	return ballot.NewINIT(
		ballot.NewINITFact(m.Height()+1, base.Round(0), m.Hash()),
		n,
		avp,
		nil,
		pk, networkID,
	)
}

func NewINITBallotWithVoteproof(
	n base.Address,
	db storage.Database,
	voteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (base.INITBallot, error) {
	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash
	var avp base.Voteproof
	switch voteproof.Stage() {
	case base.StageINIT:
		height = voteproof.Height()
		round = voteproof.Round() + 1
		switch t := voteproof.Majority().(type) {
		case base.INITBallotFact:
			previousBlock = t.PreviousBlock()
		case base.ACCEPTBallotFact:
			previousBlock = t.NewBlock()
		}

		avp = db.LastVoteproof(base.StageACCEPT)
	case base.StageACCEPT:
		height = voteproof.Height() + 1
		round = base.Round(0)
		f, ok := voteproof.Majority().(base.ACCEPTBallotFact)
		if !ok {
			return nil,
				errors.Errorf("invalid voteproof found; should have ACCEPTBallotFact, not %T", voteproof.Majority())
		}
		previousBlock = f.NewBlock()
	}

	return ballot.NewINIT(
		ballot.NewINITFact(height, round, previousBlock),
		n,
		voteproof,
		avp,
		pk, networkID,
	)
}

func NewACCEPTBallot(
	n base.Address,
	newBlock block.Block,
	voteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (base.ACCEPTBallot, error) {
	return ballot.NewACCEPT(
		ballot.NewACCEPTFact(
			newBlock.Height(),
			newBlock.Round(),
			newBlock.Proposal(),
			newBlock.Hash(),
		),
		n,
		voteproof,
		pk, networkID,
	)
}
