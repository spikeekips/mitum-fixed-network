package basicstates

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NextINITBallotFromACCEPTVoteproof(
	st storage.Database,
	local *node.Local,
	voteproof base.Voteproof,
) (ballot.INITV0, error) {
	if voteproof.Stage() != base.StageACCEPT {
		return ballot.INITV0{}, errors.Errorf("not accept voteproof")
	} else if !voteproof.IsFinished() {
		return ballot.INITV0{}, errors.Errorf("voteproof not yet finished")
	}

	var height base.Height
	var round base.Round
	var previousBlock valuehash.Hash

	// NOTE if voteproof drew, previous block should be get from local database,
	// not from voteproof.
	if voteproof.Majority() == nil {
		height = voteproof.Height()
		round = voteproof.Round() + 1

		switch m, found, err := st.ManifestByHeight(height - 1); {
		case err != nil:
			return ballot.INITV0{}, errors.Wrap(err, "failed to get manifest")
		case !found:
			return ballot.INITV0{}, errors.Errorf("manfest, height=%d not found", height-1)
		default:
			previousBlock = m.Hash()
		}
	} else if i, ok := voteproof.Majority().(ballot.ACCEPTFact); !ok {
		return ballot.INITV0{}, errors.Errorf(
			"not ballot.ACCEPTBallotFact in voteproof.Majority(); %T", voteproof.Majority())
	} else { // NOTE agreed accept voteproof
		height = voteproof.Height() + 1
		round = base.Round(0)
		previousBlock = i.NewBlock()
	}

	var avp base.Voteproof
	if voteproof.Result() == base.VoteResultMajority {
		avp = voteproof
	} else {
		switch vp, err := st.Voteproof(voteproof.Height()-1, base.StageACCEPT); {
		case err != nil:
			return ballot.INITV0{}, errors.Wrap(err, "failed to get last voteproof")
		case vp != nil:
			avp = vp
		}
	}

	return ballot.NewINITV0(
		local.Address(),
		height,
		round,
		previousBlock,
		voteproof,
		avp,
	), nil
}

func NextINITBallotFromINITVoteproof(
	st storage.Database,
	local *node.Local,
	voteproof base.Voteproof,
) (ballot.INITV0, error) {
	if voteproof.Stage() != base.StageINIT {
		return ballot.INITV0{}, errors.Errorf("not init voteproof")
	} else if !voteproof.IsFinished() {
		return ballot.INITV0{}, errors.Errorf("voteproof not yet finished")
	}

	var avp base.Voteproof
	switch vp, err := st.Voteproof(voteproof.Height()-1, base.StageACCEPT); {
	case err != nil:
		return ballot.INITV0{}, errors.Wrap(err, "failed to get last voteproof")
	case vp != nil:
		avp = vp
	}

	var previousBlock valuehash.Hash
	switch m, found, err := st.ManifestByHeight(voteproof.Height() - 1); {
	case err != nil:
		return ballot.INITV0{}, errors.Wrap(err, "failed to get previous manifest")
	case !found:
		return ballot.INITV0{}, errors.Errorf("previous manfest, height=%d not found", voteproof.Height())
	default:
		previousBlock = m.Hash()
	}

	return ballot.NewINITV0(
		local.Address(),
		voteproof.Height(),
		voteproof.Round()+1,
		previousBlock,
		voteproof,
		avp,
	), nil
}

type BallotChecker struct {
	*logging.Logging
	ballot ballot.Ballot
	lvp    base.Voteproof
}

func NewBallotChecker(blt ballot.Ballot, lvp base.Voteproof) *BallotChecker {
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

	bh := bc.ballot.Height()
	lh := bc.lvp.Height()

	if bh < lh {
		return false, errors.Errorf("lower height than last init voteproof")
	} else if bh > lh {
		return true, nil
	}

	if bc.ballot.Round() < bc.lvp.Round() {
		return false, errors.Errorf("lower round than last init voteproof")
	}

	return true, nil
}
