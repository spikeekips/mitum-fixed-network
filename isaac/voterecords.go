package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
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
		vrs.set = nil

		voteRecordsPool.Put(vrs)
	}
)

type VoteRecords struct {
	sync.RWMutex
	facts     map[string]base.BallotFact
	votes     map[string]valuehash.Hash // {node Address: fact hash}
	ballots   map[string]base.Ballot    // {node Address: ballot}
	voteproof base.VoteproofV0
	threshold base.Threshold
	set       []string
}

func NewVoteRecords(
	height base.Height,
	round base.Round,
	stage base.Stage,
	suffrages []base.Address,
	threshold base.Threshold,
) *VoteRecords {
	vrs := voteRecordsPoolGet()
	vrs.RWMutex = sync.RWMutex{}
	vrs.facts = map[string]base.BallotFact{}
	vrs.votes = map[string]valuehash.Hash{}
	vrs.ballots = map[string]base.Ballot{}
	vrs.voteproof = base.NewVoteproofV0(
		height,
		round,
		suffrages,
		threshold.Ratio,
		stage,
	)
	vrs.threshold = threshold
	vrs.set = nil

	return vrs
}

func (vrs *VoteRecords) addBallot(blt base.Ballot) bool {
	n := blt.FactSign().Node()
	fact := blt.RawFact()

	if _, found := vrs.votes[n.String()]; found {
		return true
	}

	vrs.ballots[n.String()] = blt

	factHash := vrs.sanitizeHash(fact.Hash())
	vrs.votes[n.String()] = factHash

	if _, found := vrs.facts[factHash.String()]; !found {
		vrs.facts[factHash.String()] = fact
	}

	return false
}

func (*VoteRecords) sanitizeHash(h valuehash.Hash) valuehash.Hash {
	if _, ok := h.(valuehash.Bytes); ok {
		return h
	}

	return valuehash.NewBytes(h.Bytes())
}

// Vote votes by Ballot and keep track the vote records. If getting result is
// done, Voteproof will not be updated.
func (vrs *VoteRecords) Vote(blt base.Ballot) base.Voteproof {
	vrs.Lock()
	defer vrs.Unlock()

	vrs.voteproof = vrs.vote(blt)

	return vrs.voteproof
}

func (vrs *VoteRecords) vote(blt base.Ballot) base.VoteproofV0 {
	voteproof := &vrs.voteproof

	if vrs.addBallot(blt) {
		if voteproof.IsFinished() && !voteproof.IsClosed() {
			_ = voteproof.Close()
		}

		return *voteproof
	}

	if voteproof.IsFinished() && !voteproof.IsClosed() {
		_ = voteproof.Close()

		return *voteproof
	}

	vrs.set = append(vrs.set, vrs.sanitizeHash(blt.RawFact().Hash()).String())

	if len(vrs.set) < int(vrs.threshold.Threshold) {
		return *voteproof
	}

	result, key := base.FindMajorityFromSlice(
		vrs.threshold.Total,
		vrs.threshold.Threshold,
		vrs.set,
	)

	if result == base.VoteResultMajority {
		_ = voteproof.SetMajority(vrs.facts[key])
	}

	if result != base.VoteResultNotYet {
		_ = voteproof.SetResult(result)
		_ = vrs.finishVoteproof(voteproof)
	}

	return *voteproof
}

func (vrs *VoteRecords) finishVoteproof(voteproof *base.VoteproofV0) *base.VoteproofV0 {
	facts := make([]base.BallotFact, len(vrs.facts))
	var i int
	for k := range vrs.facts {
		facts[i] = vrs.facts[k]
		i++
	}
	voteproof.SetFacts(facts)

	votes := make([]base.SignedBallotFact, len(vrs.ballots))

	i = 0
	for k := range vrs.ballots {
		blt := vrs.ballots[k]
		votes[i] = base.NewBaseSignedBallotFact(
			blt.RawFact(),
			blt.FactSign(),
		)
		i++
	}
	voteproof.SetVotes(votes)

	return voteproof.Finish()
}
