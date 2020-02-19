package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/valuehash"
)

var VoteProofV0Hint hint.Hint = hint.MustHint(VoteProofType, "0.1")

type VoteProofV0 struct {
	height     Height
	round      Round
	threshold  Threshold
	result     VoteProofResultType
	closed     bool
	stage      Stage
	majority   Fact
	facts      map[valuehash.Hash]Fact       // key: Fact.Hash(), value: Fact
	ballots    map[Address]valuehash.Hash    // key: node Address, value: ballot hash
	votes      map[Address]VoteProofNodeFact // key: node Address, value: VoteProofNodeFact
	finishedAt time.Time
}

func (vp VoteProofV0) Hint() hint.Hint {
	return VoteProofV0Hint
}

func (vp VoteProofV0) IsFinished() bool {
	return vp.result != VoteProofNotYet
}

func (vp VoteProofV0) FinishedAt() time.Time {
	return vp.finishedAt
}

func (vp VoteProofV0) IsClosed() bool {
	return vp.closed
}

func (vp VoteProofV0) Height() Height {
	return vp.height
}

func (vp VoteProofV0) Round() Round {
	return vp.round
}

func (vp VoteProofV0) Stage() Stage {
	return vp.stage
}

func (vp VoteProofV0) Result() VoteProofResultType {
	return vp.result
}

func (vp VoteProofV0) Majority() Fact {
	return vp.majority
}

func (vp VoteProofV0) Ballots() map[Address]valuehash.Hash {
	return vp.ballots
}

func (vp VoteProofV0) Bytes() []byte {
	// TODO returns proper bytes
	return nil
}

func (vp VoteProofV0) IsValid(b []byte) error {
	if err := vp.isValidFields(b); err != nil {
		return err
	}

	if err := vp.isValidFacts(b); err != nil {
		return err
	}

	// check majority
	if len(vp.votes) < int(vp.threshold.Threshold) {
		if vp.result != VoteProofNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vp.result)
		}

		return nil
	}

	return vp.isValidCheckMajority(b)
}

func (vp VoteProofV0) isValidCheckMajority(b []byte) error {
	counts := map[valuehash.Hash]uint{}
	for _, f := range vp.votes { // nolint
		counts[f.fact]++
	}

	set := make([]uint, len(counts))
	byCount := map[uint]valuehash.Hash{}

	var index int
	for h, c := range counts {
		set[index] = c
		index++
		byCount[c] = h
	}

	var fact Fact
	var factHash valuehash.Hash
	var result VoteProofResultType
	switch index := FindMajority(vp.threshold.Total, vp.threshold.Threshold, set...); index {
	case -1:
		result = VoteProofNotYet
	case -2:
		result = VoteProofDraw
	default:
		result = VoteProofMajority
		factHash = byCount[set[index]]
		fact = vp.facts[factHash]
	}

	if vp.result != result {
		return xerrors.Errorf("result mismatch; vp.result=%s != result=%s", vp.result, result)
	}

	if fact == nil {
		if vp.majority != nil {
			return xerrors.Errorf("result should be nil, but not")
		}
	} else {
		mhash, err := vp.majority.Hash(b)
		if err != nil {
			return err
		}

		if !mhash.Equal(factHash) {
			return xerrors.Errorf("fact hash mismatch; vp.majority=%s != fact=%s", mhash, factHash)
		}
	}

	return nil
}

func (vp VoteProofV0) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vp.height,
		vp.stage,
		vp.threshold,
		vp.result,
	}, b, false); err != nil {
		return err
	}
	if vp.finishedAt.IsZero() {
		return isvalid.InvalidError.Wrapf("empty finishedAt")
	}

	if vp.result != VoteProofMajority && vp.result != VoteProofDraw {
		return isvalid.InvalidError.Wrapf("invalid result; result=%v", vp.result)
	}

	if vp.majority == nil {
		if vp.result != VoteProofDraw {
			return isvalid.InvalidError.Wrapf("empty majority, but result is not draw; result=%v", vp.result)
		}
	} else if err := vp.majority.IsValid(b); err != nil {
		return err
	}

	if len(vp.facts) < 1 {
		return isvalid.InvalidError.Wrapf("empty facts")
	}

	if len(vp.ballots) < 1 {
		return isvalid.InvalidError.Wrapf("empty ballots")
	}

	if len(vp.votes) < 1 {
		return isvalid.InvalidError.Wrapf("empty votes")
	}

	if len(vp.ballots) != len(vp.votes) {
		return isvalid.InvalidError.Wrapf("vote count does not match: ballots=%d votes=%d", len(vp.ballots), len(vp.votes))
	}

	for k := range vp.ballots {
		if _, found := vp.votes[k]; !found {
			return xerrors.Errorf("unknown node found: %v", k)
		}
	}

	return nil
}

func (vp VoteProofV0) isValidFacts(b []byte) error {
	factHashes := map[valuehash.Hash]bool{}
	for node, f := range vp.votes { // nolint
		if err := node.IsValid(b); err != nil {
			return err
		}

		if _, found := vp.facts[f.fact]; !found {
			return xerrors.Errorf("missing fact found in facts: %s", f.fact.String())
		}
		factHashes[f.fact] = true
	}

	if len(factHashes) != len(vp.facts) {
		return xerrors.Errorf("unknown facts found in facts: %d", len(vp.facts)-len(factHashes))
	}

	for k, v := range vp.facts {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b, false); err != nil {
			return err
		}
		if h, err := v.Hash(b); err != nil {
			return err
		} else if !h.Equal(k) {
			return xerrors.Errorf(
				"factHash and Fact.Hash() does not match: factHash=%v != Fact.Hash()=%v",
				k.String(), h.String(),
			)
		}
	}

	for k, v := range vp.ballots {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b, false); err != nil {
			return err
		}
	}

	return nil
}
