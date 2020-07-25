package base

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	VoteproofV0Type           = hint.MustNewType(0x01, 0x30, "voteproof")
	VoteproofV0Hint hint.Hint = hint.MustHint(VoteproofV0Type, "0.0.1")
)

type VoteproofV0 struct {
	height         Height
	round          Round
	suffrages      []Address
	thresholdRatio ThresholdRatio
	result         VoteResultType
	closed         bool
	stage          Stage
	majority       Fact
	facts          []Fact
	votes          []VoteproofNodeFact
	finishedAt     time.Time
}

func NewVoteproofV0(
	height Height,
	round Round,
	suffrages []Address,
	thresholdRatio ThresholdRatio,
	stage Stage,
) VoteproofV0 {
	return VoteproofV0{
		height:         height,
		round:          round,
		suffrages:      suffrages,
		thresholdRatio: thresholdRatio,
		result:         VoteResultNotYet,
		stage:          stage,
	}
}

func (vp VoteproofV0) ID() string {
	return valuehash.NewSHA256(vp.Bytes()).String()
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

func (vp VoteproofV0) Suffrages() []Address {
	return vp.suffrages
}

func (vp VoteproofV0) ThresholdRatio() ThresholdRatio {
	return vp.thresholdRatio
}

func (vp VoteproofV0) Facts() []Fact {
	return vp.facts
}

func (vp *VoteproofV0) SetFacts(facts []Fact) *VoteproofV0 {
	vp.facts = facts

	return vp
}

func (vp VoteproofV0) Votes() []VoteproofNodeFact {
	return vp.votes
}

func (vp *VoteproofV0) SetVotes(votes []VoteproofNodeFact) *VoteproofV0 {
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

func (vp VoteproofV0) factsBytes() []byte {
	facts := map[string]Fact{}
	keys := make([]string, len(vp.facts))
	for i := range vp.facts {
		s := vp.facts[i].Hash().String()
		keys[i] = s
		facts[s] = vp.facts[i]
	}

	// NOTE without ordering, the bytes values will be varies.
	sort.Slice(keys, func(i, j int) bool {
		return strings.Compare(keys[i], keys[j]) < 0
	})

	bs := make([][]byte, len(keys))
	for i, a := range keys {
		bs[i] = facts[a].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) votesBytes() []byte {
	l := make([]VoteproofNodeFact, len(vp.votes))
	copy(l, vp.votes)

	// NOTE without ordering, the bytes values will be varies.
	sort.Slice(l, func(i, j int) bool {
		return bytes.Compare(l[i].Node().Bytes(), l[j].Node().Bytes()) < 0
	})

	bs := make([][]byte, len(l))
	for i := range l {
		bs[i] = l[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) suffragesBytes() []byte {
	bs := make([][]byte, len(vp.suffrages))
	for i := range vp.suffrages {
		bs[i] = vp.suffrages[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (vp VoteproofV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		vp.height.Bytes(),
		vp.round.Bytes(),
		util.Float64ToBytes(vp.thresholdRatio.Float64()),
		vp.result.Bytes(),
		vp.stage.Bytes(),
		vp.majority.Bytes(),
		vp.factsBytes(),
		vp.votesBytes(),
		vp.suffragesBytes(),
		[]byte(localtime.RFC3339(vp.finishedAt)),
	)
}

func (vp VoteproofV0) IsValid(networkID []byte) error {
	if err := vp.isValidFields(networkID); err != nil {
		return err
	}

	if err := vp.isValidFacts(networkID); err != nil {
		return err
	}

	// check majority
	if t, err := NewThreshold(uint(len(vp.suffrages)), vp.thresholdRatio); err != nil {
		return err
	} else if len(vp.votes) < int(t.Threshold) {
		if vp.result != VoteResultNotYet {
			return xerrors.Errorf("result should be not-yet: %s", vp.result)
		}

		return nil
	}

	return vp.isValidCheckMajority()
}

func (vp VoteproofV0) isValidCheckMajority() error {
	var threshold Threshold
	if t, err := NewThreshold(uint(len(vp.suffrages)), vp.thresholdRatio); err != nil {
		return err
	} else {
		threshold = t
	}

	counts := map[string]uint{}
	for i := range vp.votes {
		counts[vp.votes[i].fact.String()]++
	}

	set := make([]uint, len(counts))
	byCount := map[uint]string{}

	var index int
	for h, c := range counts {
		set[index] = c
		index++
		byCount[c] = h
	}

	var fact Fact
	var factHash string
	var result VoteResultType
	switch index := FindMajority(threshold.Total, threshold.Threshold, set...); index {
	case -1:
		result = VoteResultNotYet
	case -2:
		result = VoteResultDraw
	default:
		result = VoteResultMajority
		factHash = byCount[set[index]]

		for _, f := range vp.facts {
			if factHash == f.Hash().String() {
				fact = f
				break
			}
		}
	}

	if vp.result != result {
		return xerrors.Errorf("result mismatch; vp.result=%s != result=%s", vp.result, result)
	}

	if fact == nil {
		if vp.majority != nil {
			return xerrors.Errorf("result should be nil, but not")
		}
	} else {
		mh := vp.majority.Hash().String()
		if mh != factHash {
			return xerrors.Errorf("fact hash mismatch; vp.majority=%s != fact=%s", mh, factHash)
		}
	}

	return nil
}

func (vp VoteproofV0) isValidFields(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vp.height,
		vp.stage,
		vp.thresholdRatio,
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

	if len(vp.votes) < 1 {
		return isvalid.InvalidError.Errorf("empty votes")
	}

	return nil
}

func (vp VoteproofV0) isValidFacts(b []byte) error {
	factHashes := map[string]bool{}
	for i := range vp.votes {
		nf := vp.votes[i]

		if err := nf.Node().IsValid(b); err != nil {
			return err
		}

		if err := isvalid.Check([]isvalid.IsValider{nf}, b, false); err != nil {
			return err
		}

		var found bool
		for _, f := range vp.facts {
			if nf.fact.Equal(f.Hash()) {
				found = true
				break
			}
		}

		if !found {
			return xerrors.Errorf("missing fact found in facts: %s", nf.fact.String())
		}
		factHashes[nf.fact.String()] = true
	}

	if len(factHashes) != len(vp.facts) {
		return xerrors.Errorf("unknown facts found in facts: %d", len(vp.facts)-len(factHashes))
	}

	for _, f := range vp.facts {
		if err := isvalid.Check([]isvalid.IsValider{f}, b, false); err != nil {
			return err
		}
	}

	return nil
}
