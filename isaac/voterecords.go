package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
)

type VoteRecords struct {
	sync.RWMutex
	facts     map[valuehash.Hash]base.Fact
	votes     map[base.Address]valuehash.Hash // key: node Address, value: fact hash
	ballots   map[base.Address]Ballot
	voteproof base.VoteproofV0
}

func NewVoteRecords(ballot Ballot, threshold base.Threshold) *VoteRecords {
	return &VoteRecords{
		facts:   map[valuehash.Hash]base.Fact{},
		votes:   map[base.Address]valuehash.Hash{},
		ballots: map[base.Address]Ballot{},
		voteproof: base.NewVoteproofV0(
			ballot.Height(),
			ballot.Round(),
			threshold,
			ballot.Stage(),
		),
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

	return false
}

// Vote votes by Ballot and keep track the vote records. If getting result is
// done, Voteproof will not be updated.
func (vrs *VoteRecords) Vote(ballot Ballot) base.Voteproof {
	vrs.Lock()
	defer vrs.Unlock()

	vp := &vrs.voteproof
	if !vrs.vote(ballot, vp) {
		vrs.voteproof = *vp

		return vrs.voteproof
	}

	{
		facts := map[valuehash.Hash]base.Fact{}
		for k, v := range vrs.facts {
			facts[k] = v
		}
		vp.SetFacts(facts)
	}

	{
		ballots := map[base.Address]valuehash.Hash{}
		for k, v := range vrs.ballots {
			ballots[k] = v.Hash()
		}
		vp.SetBallots(ballots)
	}

	{
		votes := map[base.Address]base.VoteproofNodeFact{}
		for node, ballot := range vrs.ballots {
			votes[node] = base.NewVoteproofNodeFact(
				ballot.FactHash(),
				ballot.FactSignature(),
				ballot.Signer(),
			)
		}
		vp.SetVotes(votes)
	}

	_ = vp.Finish()

	vrs.voteproof = *vp

	return vrs.voteproof
}

func (vrs *VoteRecords) vote(ballot Ballot, voteproof *base.VoteproofV0) bool {
	if vrs.addBallot(ballot) {
		return false
	}

	if voteproof.IsFinished() {
		_ = voteproof.Close()

		return false
	} else if len(vrs.votes) < int(voteproof.Threshold().Threshold) {
		return false
	}

	set := make([]string, len(vrs.votes))
	facts := map[string]base.Fact{}

	var i int
	for n := range vrs.votes {
		factHash := vrs.votes[n]
		key := factHash.String()
		set[i] = key
		facts[key] = vrs.facts[factHash]
		i++
	}

	var fact base.Fact
	var result base.VoteResultType

	threshold := voteproof.Threshold()
	result, key := base.FindMajorityFromSlice(threshold.Total, threshold.Threshold, set)
	if result == base.VoteResultMajority {
		fact = facts[key]
	}

	_ = voteproof.SetResult(result).
		SetMajority(fact)

	return true
}
