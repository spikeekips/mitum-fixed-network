package isaac

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
	vp        VoteProofV0
}

func NewVoteRecords(ballot Ballot, threshold Threshold) *VoteRecords {
	return &VoteRecords{
		facts:     map[valuehash.Hash]Fact{},
		votes:     map[Address]valuehash.Hash{},
		factCount: map[valuehash.Hash]uint{},
		ballots:   map[Address]Ballot{},
		vp: VoteProofV0{
			height:    ballot.Height(),
			round:     ballot.Round(),
			stage:     ballot.Stage(),
			threshold: threshold,
			result:    VoteProofNotYet,
			facts:     map[valuehash.Hash]Fact{},
			ballots:   map[Address]valuehash.Hash{},
			votes:     map[Address]VoteProofNodeFact{},
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
// done, VoteProof will not be updated.
func (vrs *VoteRecords) Vote(ballot Ballot) VoteProof {
	vrs.Lock()
	defer vrs.Unlock()

	if !vrs.vote(ballot) {
		return vrs.vp
	}

	{
		facts := map[valuehash.Hash]Fact{}
		for k, v := range vrs.facts {
			facts[k] = v
		}
		vrs.vp.facts = facts
	}

	{
		ballots := map[Address]valuehash.Hash{}
		for k, v := range vrs.ballots {
			ballots[k] = v.Hash()
		}
		vrs.vp.ballots = ballots
	}

	{
		votes := map[Address]VoteProofNodeFact{}
		for node, ballot := range vrs.ballots {
			votes[node] = VoteProofNodeFact{
				fact:          ballot.FactHash(),
				factSignature: ballot.FactSignature(),
				signer:        ballot.Signer(),
			}
		}
		vrs.vp.votes = votes
	}

	return vrs.vp
}

func (vrs *VoteRecords) vote(ballot Ballot) bool {
	if vrs.addBallot(ballot) {
		return false
	}

	if vrs.vp.IsFinished() {
		vrs.vp.closed = true
		return false
	} else if len(vrs.votes) < int(vrs.vp.threshold.Threshold) {
		return false
	}

	byCount := map[uint]Fact{}
	set := make([]uint, len(vrs.factCount))
	for factHash, c := range vrs.factCount {
		set = append(set, c)
		byCount[c] = vrs.facts[factHash]
	}

	if len(set) > 0 {
		sort.Slice(set, func(i, j int) bool { return set[i] > set[j] })
	}

	var fact Fact
	var result VoteProofResultType
	switch index := FindMajority(vrs.vp.threshold.Total, vrs.vp.threshold.Threshold, set...); index {
	case -1:
		result = VoteProofNotYet
	case -2:
		result = VoteProofDraw
	default:
		result = VoteProofMajority
		fact = byCount[set[index]]
	}

	vrs.vp.result = result
	vrs.vp.majority = fact

	return true
}
