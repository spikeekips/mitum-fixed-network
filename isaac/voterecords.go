package isaac

import (
	"sort"
	"sync"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/valuehash"
)

type VoteRecords struct {
	sync.RWMutex
	facts     map[valuehash.Hash]operation.Fact
	votes     map[Address]valuehash.Hash // key: node Address, value: fact hash
	factCount map[valuehash.Hash]uint
	ballots   map[Address]Ballot
	voteproof VoteproofV0
}

func NewVoteRecords(ballot Ballot, threshold Threshold) *VoteRecords {
	return &VoteRecords{
		facts:     map[valuehash.Hash]operation.Fact{},
		votes:     map[Address]valuehash.Hash{},
		factCount: map[valuehash.Hash]uint{},
		ballots:   map[Address]Ballot{},
		voteproof: VoteproofV0{
			height:    ballot.Height(),
			round:     ballot.Round(),
			stage:     ballot.Stage(),
			threshold: threshold,
			result:    VoteproofNotYet,
			facts:     map[valuehash.Hash]operation.Fact{},
			ballots:   map[Address]valuehash.Hash{},
			votes:     map[Address]VoteproofNodeFact{},
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
// done, Voteproof will not be updated.
func (vrs *VoteRecords) Vote(ballot Ballot) Voteproof {
	vrs.Lock()
	defer vrs.Unlock()

	if !vrs.vote(ballot) {
		return vrs.voteproof
	}

	{
		facts := map[valuehash.Hash]operation.Fact{}
		for k, v := range vrs.facts {
			facts[k] = v
		}
		vrs.voteproof.facts = facts
	}

	{
		ballots := map[Address]valuehash.Hash{}
		for k, v := range vrs.ballots {
			ballots[k] = v.Hash()
		}
		vrs.voteproof.ballots = ballots
	}

	{
		votes := map[Address]VoteproofNodeFact{}
		for node, ballot := range vrs.ballots {
			votes[node] = VoteproofNodeFact{
				fact:          ballot.FactHash(),
				factSignature: ballot.FactSignature(),
				signer:        ballot.Signer(),
			}
		}
		vrs.voteproof.votes = votes
	}

	vrs.voteproof.finishedAt = localtime.Now()

	return vrs.voteproof
}

func (vrs *VoteRecords) vote(ballot Ballot) bool {
	if vrs.addBallot(ballot) {
		return false
	}

	if vrs.voteproof.IsFinished() {
		vrs.voteproof.closed = true
		return false
	} else if len(vrs.votes) < int(vrs.voteproof.threshold.Threshold) {
		return false
	}

	byCount := map[uint]operation.Fact{}
	set := make([]uint, len(vrs.factCount))
	for factHash, c := range vrs.factCount {
		set = append(set, c)
		byCount[c] = vrs.facts[factHash]
	}

	if len(set) > 0 {
		sort.Slice(set, func(i, j int) bool { return set[i] > set[j] })
	}

	var fact operation.Fact
	var result VoteproofResultType
	switch index := FindMajority(vrs.voteproof.threshold.Total, vrs.voteproof.threshold.Threshold, set...); index {
	case -1:
		result = VoteproofNotYet
	case -2:
		result = VoteproofDraw
	default:
		result = VoteproofMajority
		fact = byCount[set[index]]
	}

	vrs.voteproof.result = result
	vrs.voteproof.majority = fact

	return true
}
