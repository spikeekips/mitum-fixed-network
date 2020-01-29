package mitum

import (
	"sort"
	"sync"

	"github.com/spikeekips/mitum/valuehash"
)

type VoteRecords struct {
	sync.RWMutex
	facts     map[valuehash.Hash]Fact
	votes     map[Address]valuehash.Hash // key: node Address, value: fact hash
	factCount map[valuehash.Hash]uint
	ballots   map[Address]Ballot
	vr        VoteResult
}

func NewVoteRecords(ballot Ballot, threshold Threshold) *VoteRecords {
	return &VoteRecords{
		facts:     map[valuehash.Hash]Fact{},
		votes:     map[Address]valuehash.Hash{},
		factCount: map[valuehash.Hash]uint{},
		ballots:   map[Address]Ballot{},
		vr: VoteResult{
			height:    ballot.Height(),
			round:     ballot.Round(),
			stage:     ballot.Stage(),
			threshold: threshold,
			result:    VoteResultNotYet,
			facts:     map[valuehash.Hash]Fact{},
			ballots:   map[Address]valuehash.Hash{},
			votes:     map[Address]valuehash.Hash{},
		},
	}
}

func (vrs *VoteRecords) addBallot(ballot Ballot) bool {
	if _, found := vrs.votes[ballot.Node()]; found {
		return true
	}

	vrs.ballots[ballot.Node()] = ballot

	factHash := ballot.FactHash()
	vrs.votes[ballot.Node()] = factHash

	if _, found := vrs.facts[factHash]; !found {
		vrs.facts[factHash] = ballot.Fact()
	}
	vrs.factCount[factHash]++

	return false
}

// Vote votes by Ballot and keep track the vote records. If getting result is
// done, VoteResult will not be updated.
func (vrs *VoteRecords) Vote(ballot Ballot) VoteResult {
	vrs.Lock()
	defer vrs.Unlock()

	if !vrs.vote(ballot) {
		return vrs.vr
	}

	{
		facts := map[valuehash.Hash]Fact{}
		for k, v := range vrs.facts {
			facts[k] = v
		}
		vrs.vr.facts = facts
	}

	{
		ballots := map[Address]valuehash.Hash{}
		for k, v := range vrs.ballots {
			ballots[k] = v.Hash()
		}
		vrs.vr.ballots = ballots
	}

	{
		votes := map[Address]valuehash.Hash{}
		for k, v := range vrs.votes {
			votes[k] = v
		}
		vrs.vr.votes = votes
	}

	return vrs.vr
}

func (vrs *VoteRecords) vote(ballot Ballot) bool {
	if vrs.addBallot(ballot) {
		return false
	}

	if vrs.vr.IsFinished() {
		return false
	} else if len(vrs.votes) < int(vrs.vr.threshold.Threshold) {
		return false
	}

	byCount := map[uint]Fact{}
	var set []uint
	for factHash, c := range vrs.factCount {
		set = append(set, c)
		byCount[c] = vrs.facts[factHash]
	}

	if len(set) > 0 {
		sort.Slice(set, func(i, j int) bool { return set[i] > set[j] })
	}

	var fact Fact
	var result VoteResultType
	switch index := FindMajority(vrs.vr.threshold.Total, vrs.vr.threshold.Threshold, set...); index {
	case -1:
		result = VoteResultNotYet
	case -2:
		result = VoteResultDraw
	default:
		result = VoteResultMajority
		fact = byCount[set[index]]
	}

	vrs.vr.result = result
	vrs.vr.majority = fact

	return true
}
