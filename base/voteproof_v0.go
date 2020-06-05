package base

import (
	"bytes"
	"sort"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	VoteproofV0Type           = hint.MustNewType(0x04, 0x00, "voteproof")
	VoteproofV0Hint hint.Hint = hint.MustHint(VoteproofV0Type, "0.0.1")
)

type VoteproofV0 struct {
	height     Height
	round      Round
	threshold  Threshold
	result     VoteResultType
	closed     bool
	stage      Stage
	majority   Fact
	facts      map[valuehash.Hash]Fact       // key: Fact.Hash(), value: Fact
	ballots    map[Address]valuehash.Hash    // key: node Address, value: ballot hash
	votes      map[Address]VoteproofNodeFact // key: node Address, value: VoteproofNodeFact
	finishedAt time.Time
}

func NewVoteproofV0(
	height Height,
	round Round,
	threshold Threshold,
	stage Stage,
) VoteproofV0 {
	return VoteproofV0{
		height:    height,
		round:     round,
		threshold: threshold,
		result:    VoteResultNotYet,
		stage:     stage,
		facts:     map[valuehash.Hash]Fact{},
		ballots:   map[Address]valuehash.Hash{},
		votes:     map[Address]VoteproofNodeFact{},
	}
}

func (vp VoteproofV0) Hint() hint.Hint {
	return VoteproofV0Hint
}

func (vp VoteproofV0) IsFinished() bool {
	return vp.result != VoteResultNotYet
}

func (vp VoteproofV0) FinishedAt() time.Time {
	return vp.finishedAt
}

func (vp VoteproofV0) IsClosed() bool {
	return vp.closed
}

func (vp VoteproofV0) Height() Height {
	return vp.height
}

func (vp VoteproofV0) Round() Round {
	return vp.round
}

func (vp VoteproofV0) Stage() Stage {
	return vp.stage
}

func (vp VoteproofV0) Result() VoteResultType {
	return vp.result
}

func (vp *VoteproofV0) SetResult(result VoteResultType) *VoteproofV0 {
	vp.result = result

	return vp
}

func (vp VoteproofV0) Majority() Fact {
	return vp.majority
}

func (vp *VoteproofV0) SetMajority(fact Fact) *VoteproofV0 {
	vp.majority = fact

	return vp
}

func (vp VoteproofV0) Ballots() map[Address]valuehash.Hash {
	return vp.ballots
}

func (vp *VoteproofV0) SetBallots(ballots map[Address]valuehash.Hash) *VoteproofV0 {
	vp.ballots = ballots

	return vp
}

func (vp VoteproofV0) Threshold() Threshold {
	return vp.threshold
}

func (vp VoteproofV0) Facts() map[valuehash.Hash]Fact {
	return vp.facts
}

func (vp *VoteproofV0) SetFacts(facts map[valuehash.Hash]Fact) *VoteproofV0 {
	vp.facts = facts

	return vp
}

func (vp VoteproofV0) Votes() map[Address]VoteproofNodeFact {
	return vp.votes
}

func (vp *VoteproofV0) SetVotes(votes map[Address]VoteproofNodeFact) *VoteproofV0 {
	vp.votes = votes

	return vp
}

func (vp *VoteproofV0) Finish() *VoteproofV0 {
	vp.finishedAt = localtime.Now()

	return vp
}

func (vp *VoteproofV0) Close() *VoteproofV0 {
	vp.closed = true

	return vp
}

func (vp VoteproofV0) ballotsBytes() []byte {
	keys := make([]Address, len(vp.ballots))
	var i int
	for a := range vp.ballots {
		keys[i] = a
		i++
	}

	// NOTE without ordering, the bytes values will be varies.
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(
			keys[i].Bytes(),
			keys[j].Bytes(),
		) < 0
	})

	bs := make([][]byte, len(keys))
	for i, a := range keys {
		bs[i] = vp.ballots[a].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) factsBytes() []byte {
	keys := make([]valuehash.Hash, len(vp.facts))
	var i int
	for a := range vp.facts {
		keys[i] = a
		i++
	}

	// NOTE without ordering, the bytes values will be varies.
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(
			keys[i].Bytes(),
			keys[j].Bytes(),
		) < 0
	})

	bs := make([][]byte, len(keys))
	for i, a := range keys {
		bs[i] = vp.facts[a].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) votesBytes() []byte {
	keys := make([]Address, len(vp.votes))
	var i int
	for a := range vp.votes {
		keys[i] = a
		i++
	}

	// NOTE without ordering, the bytes values will be varies.
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(
			keys[i].Bytes(),
			keys[j].Bytes(),
		) < 0
	})

	bs := make([][]byte, len(keys))
	for i, a := range keys {
		bs[i] = vp.votes[a].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		vp.height.Bytes(),
		vp.round.Bytes(),
		vp.threshold.Bytes(),
		vp.result.Bytes(),
		vp.stage.Bytes(),
		vp.majority.Bytes(),
		vp.ballotsBytes(),
		vp.factsBytes(),
		vp.votesBytes(),
		[]byte(localtime.RFC3339(vp.finishedAt)),
	)
}

func (vp VoteproofV0) IsValid(b []byte) error {
	if err := vp.isValidFields(b); err != nil {
		return err
	}

	if err := vp.isValidFacts(b); err != nil {
		return err
	}

	// check majority
	if len(vp.votes) < int(vp.threshold.Threshold) {
		if vp.result != VoteResultNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vp.result)
		}

		return nil
	}

	return vp.isValidCheckMajority()
}

func (vp VoteproofV0) isValidCheckMajority() error {
	counts := map[valuehash.Hash]uint{}
	for a := range vp.votes {
		counts[vp.votes[a].fact]++
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
	var result VoteResultType
	switch index := FindMajority(vp.threshold.Total, vp.threshold.Threshold, set...); index {
	case -1:
		result = VoteResultNotYet
	case -2:
		result = VoteResultDraw
	default:
		result = VoteResultMajority
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
		mhash := vp.majority.Hash()
		if !mhash.Equal(factHash) {
			return xerrors.Errorf("fact hash mismatch; vp.majority=%s != fact=%s", mhash, factHash)
		}
	}

	return nil
}

func (vp VoteproofV0) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vp.height,
		vp.stage,
		vp.threshold,
		vp.result,
	}, nil, false); err != nil {
		return err
	}
	if vp.finishedAt.IsZero() {
		return isvalid.InvalidError.Errorf("empty finishedAt")
	}

	if vp.result != VoteResultMajority && vp.result != VoteResultDraw {
		return isvalid.InvalidError.Errorf("invalid result; result=%v", vp.result)
	}

	if vp.majority == nil {
		if vp.result != VoteResultDraw {
			return isvalid.InvalidError.Errorf("empty majority, but result is not draw; result=%v", vp.result)
		}
	} else if err := vp.majority.IsValid(b); err != nil {
		return err
	}

	if len(vp.facts) < 1 {
		return isvalid.InvalidError.Errorf("empty facts")
	}

	if len(vp.ballots) < 1 {
		return isvalid.InvalidError.Errorf("empty ballots")
	}

	if len(vp.votes) < 1 {
		return isvalid.InvalidError.Errorf("empty votes")
	}

	if len(vp.ballots) != len(vp.votes) {
		return isvalid.InvalidError.Errorf("vote count does not match: ballots=%d votes=%d", len(vp.ballots), len(vp.votes))
	}

	for k := range vp.ballots {
		if _, found := vp.votes[k]; !found {
			return xerrors.Errorf("unknown node found: %v", k)
		}
	}

	return nil
}

func (vp VoteproofV0) isValidFacts(b []byte) error {
	factHashes := map[valuehash.Hash]bool{}
	for node := range vp.votes {
		if err := node.IsValid(b); err != nil {
			return err
		}

		f := vp.votes[node]
		if err := isvalid.Check([]isvalid.IsValider{f}, b, false); err != nil {
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
		if h := v.Hash(); !h.Equal(k) {
			return xerrors.Errorf(
				"factHash and Fact.Hash() does not match: factHash=%v != Fact.Hash()=%v",
				k.String(), h.String(),
			)
		}
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b, false); err != nil {
			return err
		}
	}

	for k, v := range vp.ballots {
		if err := isvalid.Check([]isvalid.IsValider{k, v}, b, false); err != nil {
			return err
		}
	}

	return nil
}
