package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/valuehash"
)

var voteRecordsPool = sync.Pool{
	New: func() interface{} {
		return new(VoteRecords)
	},
}

var (
	voteRecordsPoolGet = func() *VoteRecords {
		return voteRecordsPool.Get().(*VoteRecords)
	}
	voteRecordsPoolPut = func(vrs *VoteRecords) {
		vrs.Lock()
		defer vrs.Unlock()

		vrs.facts = nil
		vrs.votes = nil
		vrs.ballots = nil
		vrs.voteproof = base.VoteproofV0{}
		vrs.threshold = base.Threshold{}

		voteRecordsPool.Put(vrs)
	}
)

type VoteRecords struct {
	sync.RWMutex
	facts     map[string]base.Fact
	votes     map[string]valuehash.Hash // {node Address: fact hash}
	ballots   map[string]ballot.Ballot  // {node Address: ballot}
	voteproof base.VoteproofV0
	threshold base.Threshold
}

func NewVoteRecords(blt ballot.Ballot, suffrages []base.Address, threshold base.Threshold) *VoteRecords {
	vr := voteRecordsPoolGet()
	vr.RWMutex = sync.RWMutex{}
	vr.facts = map[string]base.Fact{}
	vr.votes = map[string]valuehash.Hash{}
	vr.ballots = map[string]ballot.Ballot{}
	vr.voteproof = base.NewVoteproofV0(
		blt.Height(),
		blt.Round(),
		suffrages,
		threshold.Ratio,
		blt.Stage(),
	)
	vr.threshold = threshold

	return vr
}

func (vrs *VoteRecords) addBallot(blt ballot.Ballot) bool {
	if _, found := vrs.votes[blt.Node().String()]; found {
		return true
	}

	vrs.ballots[blt.Node().String()] = blt

	factHash := vrs.sanitizeHash(blt.Fact().Hash())
	vrs.votes[blt.Node().String()] = factHash

	if _, found := vrs.facts[factHash.String()]; !found {
		vrs.facts[factHash.String()] = blt.Fact()
	}

	return false
}

func (vrs *VoteRecords) sanitizeHash(h valuehash.Hash) valuehash.Hash {
	if _, ok := h.(valuehash.Bytes); ok {
		return h
	}

	return valuehash.NewBytes(h.Bytes())
}

// Vote votes by Ballot and keep track the vote records. If getting result is
// done, Voteproof will not be updated.
func (vrs *VoteRecords) Vote(blt ballot.Ballot) base.Voteproof {
	vrs.Lock()
	defer vrs.Unlock()

	vp := &vrs.voteproof
	if !vrs.vote(blt, vp) {
		vrs.voteproof = *vp

		return vrs.voteproof
	}

	{
		facts := make([]base.Fact, len(vrs.facts))
		var i int
		for _, v := range vrs.facts {
			facts[i] = v
			i++
		}
		vp.SetFacts(facts)
	}

	{
		votes := make([]base.VoteproofNodeFact, len(vrs.ballots))

		var i int
		for _, blt := range vrs.ballots {
			votes[i] = base.NewVoteproofNodeFact(
				blt.Node(),
				vrs.sanitizeHash(blt.Hash()),
				vrs.sanitizeHash(blt.Fact().Hash()),
				blt.FactSignature(),
				blt.Signer(),
			)
			i++
		}
		vp.SetVotes(votes)
	}

	_ = vp.Finish()

	vrs.voteproof = *vp

	return vrs.voteproof
}

func (vrs *VoteRecords) vote(blt ballot.Ballot, voteproof *base.VoteproofV0) bool {
	if vrs.addBallot(blt) {
		if voteproof.IsFinished() && !voteproof.IsClosed() {
			_ = voteproof.Close()
		}

		return false
	}

	if voteproof.IsFinished() && !voteproof.IsClosed() {
		_ = voteproof.Close()

		return false
	} else if len(vrs.votes) < int(vrs.threshold.Threshold) {
		return false
	}

	set := make([]string, len(vrs.votes))
	facts := map[string]base.Fact{}

	var i int
	for n := range vrs.votes {
		key := vrs.votes[n].String()
		set[i] = key
		facts[key] = vrs.facts[key]
		i++
	}

	var fact base.Fact
	var result base.VoteResultType

	result, key := base.FindMajorityFromSlice(vrs.threshold.Total, vrs.threshold.Threshold, set)
	if result == base.VoteResultMajority {
		fact = facts[key]
	}

	_ = voteproof.SetResult(result).
		SetMajority(fact)

	return true
}
