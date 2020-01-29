package mitum

import (
	"encoding/json"
	"sort"
	"sync"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	InvalidVoteResultTypeError = errors.NewError("invalid VoteResultType")
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

	return InvalidVoteResultTypeError.Wrapf("VoteResultType=%d", vrt)
}

func (vrt VoteResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

type VoteResult struct {
	sync.RWMutex
	height    Height
	round     Round
	stage     Stage
	facts     map[valuehash.Hash]Fact
	votes     map[Address]valuehash.Hash // key: node Address, value: fact hash
	factCount map[valuehash.Hash]uint
	threshold Threshold
	result    VoteResultType
	majority  Fact
	ballots   map[Address]Ballot
}

func NewVoteResult(ballot Ballot, threshold Threshold) *VoteResult {
	return &VoteResult{
		height:    ballot.Height(),
		round:     ballot.Round(),
		stage:     ballot.Stage(),
		facts:     map[valuehash.Hash]Fact{},
		votes:     map[Address]valuehash.Hash{},
		factCount: map[valuehash.Hash]uint{},
		threshold: threshold,
		result:    VoteResultNotYet,
		ballots:   map[Address]Ballot{},
	}
}

func (vr *VoteResult) String() string {
	b, _ := json.Marshal(vr)

	return string(b)
}

func (vr *VoteResult) Height() Height {
	return vr.height
}

func (vr *VoteResult) Round() Round {
	return vr.round
}

func (vr *VoteResult) Stage() Stage {
	return vr.stage
}

func (vr *VoteResult) Bytes() []byte {
	return nil
}

func (vr *VoteResult) Result() VoteResultType {
	vr.RLock()
	defer vr.RUnlock()

	return vr.result
}

func (vr *VoteResult) Majority() Fact {
	vr.RLock()
	defer vr.RUnlock()

	return vr.majority
}

func (vr *VoteResult) IsVoted(node Address) bool {
	vr.RLock()
	defer vr.RUnlock()

	_, found := vr.votes[node]

	return found
}

func (vr *VoteResult) VoteCount() int {
	vr.RLock()
	defer vr.RUnlock()

	return len(vr.votes)
}

func (vr *VoteResult) IsFinished() bool {
	vr.RLock()
	defer vr.RUnlock()

	return vr.isFinished()
}

func (vr *VoteResult) isFinished() bool {
	return vr.result != VoteResultNotYet
}

func (vr *VoteResult) addBallot(ballot Ballot) bool {
	if _, found := vr.votes[ballot.Node()]; found {
		return true
	}

	vr.ballots[ballot.Node()] = ballot

	factHash := ballot.FactHash()
	vr.votes[ballot.Node()] = factHash

	if _, found := vr.facts[factHash]; !found {
		vr.facts[factHash] = ballot.Fact()
	}
	vr.factCount[factHash]++

	return false
}

func (vr *VoteResult) Vote(ballot Ballot) (VoteResultType, Fact) {
	vr.Lock()
	defer vr.Unlock()

	if vr.addBallot(ballot) {
		return vr.result, nil
	}

	if vr.isFinished() {
		return vr.result, nil
	} else if len(vr.votes) < int(vr.threshold.Threshold) {
		return vr.result, nil
	}

	byCount := map[uint]Fact{}
	var set []uint
	for factHash, c := range vr.factCount {
		set = append(set, c)
		byCount[c] = vr.facts[factHash]
	}

	if len(set) > 0 {
		sort.Slice(set, func(i, j int) bool { return set[i] > set[j] })
	}

	var fact Fact
	var result VoteResultType
	switch index := FindMajority(vr.threshold.Total, vr.threshold.Threshold, set...); index {
	case -1:
		result = VoteResultNotYet
	case -2:
		result = VoteResultDraw
	default:
		result = VoteResultMajority
		fact = byCount[set[index]]
	}

	vr.result = result
	vr.majority = fact

	return vr.result, vr.majority
}
