package basicstates

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NextINITBallotFromACCEPTVoteproof(
	db storage.Database,
	local node.Local,
	voteproof base.Voteproof,
	networkID base.NetworkID,
) (base.INITBallot, error) {
	if voteproof.Stage() != base.StageACCEPT {
		return nil, errors.Errorf("not accept voteproof")
	} else if !voteproof.IsFinished() {
		return nil, errors.Errorf("voteproof not yet finished")
	}

	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash

	// NOTE if voteproof drew, previous block should be get from local database,
	// not from voteproof.
	if voteproof.Majority() == nil {
		height = voteproof.Height()
		round = voteproof.Round() + 1

		switch m, found, err := db.ManifestByHeight(height - 1); {
		case err != nil:
			return nil, errors.Wrap(err, "failed to get manifest")
		case !found:
			return nil, errors.Errorf("manfest, height=%d not found", height-1)
		default:
			previousBlock = m.Hash()
		}
	} else if i, ok := voteproof.Majority().(base.ACCEPTBallotFact); !ok {
		return nil, errors.Errorf(
			"not ballot.ACCEPTBallotFact in voteproof.Majority(); %T", voteproof.Majority())
	} else { // NOTE agreed accept voteproof
		height = voteproof.Height() + 1
		round = base.Round(0)
		previousBlock = i.NewBlock()
	}

	var avp base.Voteproof
	if voteproof.Result() != base.VoteResultMajority {
		switch vp, err := db.Voteproof(voteproof.Height()-1, base.StageACCEPT); {
		case err != nil:
			return nil, errors.Wrap(err, "failed to get last voteproof")
		case vp != nil:
			avp = vp
		}
	}

	return ballot.NewINIT(
		ballot.NewINITFact(
			height,
			round,
			previousBlock,
		),
		local.Address(),
		voteproof,
		avp,
		local.Privatekey(), networkID,
	)
}

func NextINITBallotFromINITVoteproof(
	db storage.Database,
	local node.Local,
	voteproof,
	acceptVoteproof base.Voteproof,
	networkID base.NetworkID,
) (base.INITBallot, error) {
	if voteproof.Stage() != base.StageINIT {
		return nil, errors.Errorf("not init voteproof")
	} else if !voteproof.IsFinished() {
		return nil, errors.Errorf("voteproof not yet finished")
	}

	if acceptVoteproof == nil {
		switch vp, err := db.Voteproof(voteproof.Height()-1, base.StageACCEPT); {
		case err != nil:
			return nil, errors.Wrap(err, "failed to get last voteproof")
		case vp != nil:
			acceptVoteproof = vp
		}
	}

	var previousBlock valuehash.Hash
	switch m, found, err := db.ManifestByHeight(voteproof.Height() - 1); {
	case err != nil:
		return nil, errors.Wrap(err, "failed to get previous manifest")
	case !found:
		return nil, errors.Errorf("previous manfest, height=%d not found", voteproof.Height())
	default:
		previousBlock = m.Hash()
	}

	return ballot.NewINIT(
		ballot.NewINITFact(
			voteproof.Height(),
			voteproof.Round()+1,
			previousBlock,
		),
		local.Address(),
		voteproof,
		acceptVoteproof,
		local.Privatekey(), networkID,
	)
}

type BallotChecker struct {
	*logging.Logging
	ballot base.Ballot
	lvp    base.Voteproof
}

func NewBallotChecker(blt base.Ballot, lvp base.Voteproof) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballot-checker-in-states")
		}),
		ballot: blt,
		lvp:    lvp,
	}
}

func (bc *BallotChecker) CheckWithLastVoteproof() (bool, error) {
	if bc.lvp == nil {
		return true, nil
	}

	fact := bc.ballot.RawFact()
	bh := fact.Height()
	lh := bc.lvp.Height()

	if bh < lh {
		return false, errors.Errorf("lower height than last init voteproof")
	} else if bh > lh {
		return true, nil
	}

	if fact.Round() < bc.lvp.Round() {
		return false, errors.Errorf("lower round than last init voteproof")
	}

	return true, nil
}

func signBallot(blt base.Ballot, priv key.Privatekey, networkID base.NetworkID) error {
	var signer seal.Signer
	switch t := blt.(type) {
	case ballot.INIT:
		signer = (interface{})(&t).(seal.Signer)
	case ballot.Proposal:
		signer = (interface{})(&t).(seal.Signer)
	case ballot.ACCEPT:
		signer = (interface{})(&t).(seal.Signer)
	default:
		return errors.Errorf("failed to sign ballot; unknown ballot type; %T", blt)
	}

	return signer.Sign(priv, networkID)
}

func signBallotWithFact(blt base.Ballot, n base.Address, priv key.Privatekey, networkID base.NetworkID) error {
	var signer base.SignWithFacter
	switch t := blt.(type) {
	case ballot.INIT:
		signer = (interface{})(&t).(base.SignWithFacter)
	case ballot.Proposal:
		signer = (interface{})(&t).(base.SignWithFacter)
	case ballot.ACCEPT:
		signer = (interface{})(&t).(base.SignWithFacter)
	default:
		return errors.Errorf("failed to sign ballot with fact; unknown ballot type; %T", blt)
	}

	return signer.SignWithFact(n, priv, networkID)
}
