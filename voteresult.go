package mitum

import (
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type VoteResultType uint8

const (
	VoteResultNotYet VoteResultType = iota
	VoteResultDraw
	VoteResultMajority
)

func (vrt VoteResultType) String() string {
	switch vrt {
	case VoteResultNotYet:
		return "NOT-YET"
	case VoteResultDraw:
		return "DRAW"
	case VoteResultMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteResultType>"
	}
}

func (vrt VoteResultType) IsValid([]byte) error {
	switch vrt {
	case VoteResultNotYet, VoteResultDraw, VoteResultMajority:
		return nil
	}

	return InvalidError.Wrapf("VoteResultType=%d", vrt)
}

func (vrt VoteResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

type VoteResult struct {
	height    Height
	round     Round
	stage     Stage
	threshold Threshold
	result    VoteResultType
	majority  Fact
	facts     map[valuehash.Hash]Fact    // key: Fact.Hash(), value: Fact
	ballots   map[Address]valuehash.Hash // key: node Address, value: ballot hash
	votes     map[Address]valuehash.Hash // key: node Address, value: fact hash
}

func (vr VoteResult) IsFinished() bool {
	return vr.result != VoteResultNotYet
}

func (vr VoteResult) Height() Height {
	return vr.height
}

func (vr VoteResult) Round() Round {
	return vr.round
}

func (vr VoteResult) Stage() Stage {
	return vr.stage
}

func (vr VoteResult) Result() VoteResultType {
	return vr.result
}

func (vr VoteResult) Ballots() map[Address]valuehash.Hash {
	return vr.ballots
}

func (vr VoteResult) Bytes() []byte {
	return nil
}

func (vr VoteResult) IsValid(b []byte) error {
	if err := vr.isValidFields(b); err != nil {
		return err
	}

	// check majority
	if len(vr.votes) < int(vr.threshold.Threshold) {
		if vr.result != VoteResultNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vr.result)
		}

		return nil
	}

	return vr.isValidCheckMajority(b)
}

func (vr VoteResult) isValidCheckMajority(b []byte) error {
	counts := map[valuehash.Hash]uint{}
	for _, h := range vr.votes {
		counts[h]++
	}

	var set []uint
	byCount := map[uint]valuehash.Hash{}
	for h, c := range counts {
		set = append(set, c)
		byCount[c] = h
	}

	var fact Fact
	var factHash valuehash.Hash
	var result VoteResultType
	switch index := FindMajority(vr.threshold.Total, vr.threshold.Threshold, set...); index {
	case -1:
		result = VoteResultNotYet
	case -2:
		result = VoteResultDraw
	default:
		result = VoteResultMajority
		factHash = byCount[set[index]]
		fact = vr.facts[factHash]
	}

	if vr.result != result {
		return xerrors.Errorf("result mismatch; vr.result=%s != result=%s", vr.result, result)
	}

	if fact == nil {
		if vr.majority != nil {
			return xerrors.Errorf("result should be nil, but not")
		}
	} else {
		mhash, err := vr.majority.Hash(b)
		if err != nil {
			return err
		}

		if !mhash.Equal(factHash) {
			return xerrors.Errorf("fact hash mismatch; vr.majority=%s != fact=%s", mhash, factHash)
		}
	}

	return nil
}

func (vr VoteResult) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vr.height,
		vr.stage,
		vr.threshold,
		vr.result,
	}, b); err != nil {
		return err
	}

	if vr.majority == nil {
		if vr.result == VoteResultMajority {
			return InvalidError.Wrapf("empty majority")
		}
	} else {
		if err := vr.majority.IsValid(b); err != nil {
			return err
		}
	}

	if len(vr.facts) < 1 {
		return InvalidError.Wrapf("empty facts")
	}

	if len(vr.ballots) < 1 {
		return InvalidError.Wrapf("empty ballots")
	}

	if len(vr.votes) < 1 {
		return InvalidError.Wrapf("empty votes")
	}

	if len(vr.ballots) != len(vr.votes) {
		return InvalidError.Wrapf("vote count does not match: ballots=%d votes=%d", len(vr.ballots), len(vr.votes))
	}

	for k, v := range vr.facts {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
	}

	for k, v := range vr.ballots {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
	}

	for k, v := range vr.votes {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b); err != nil {
			return err
		}
	}

	factHashes := map[valuehash.Hash]bool{}
	for _, factHash := range vr.votes {
		if _, found := vr.facts[factHash]; !found {
			return xerrors.Errorf("missing fact found in facts: %s", factHash.String())
		}
		factHashes[factHash] = true
	}

	if len(factHashes) != len(vr.facts) {
		return xerrors.Errorf("unknown facts found in facts: %d", len(vr.facts)-len(factHashes))
	}

	return nil
}
