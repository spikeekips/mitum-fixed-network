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
		vrs.set = nil

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
	set       []string
}

func NewVoteRecords(blt ballot.Ballot, suffrages []base.Address, threshold base.Threshold) *VoteRecords {
	vrs := voteRecordsPoolGet()
	vrs.RWMutex = sync.RWMutex{}
	vrs.facts = map[string]base.Fact{}
	vrs.votes = map[string]valuehash.Hash{}
	vrs.ballots = map[string]ballot.Ballot{}
	vrs.voteproof = base.NewVoteproofV0(
		blt.Height(),
		blt.Round(),
		suffrages,
		threshold.Ratio,
		blt.Stage(),
	)
	vrs.threshold = threshold
	vrs.set = nil

	return vrs
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

func (*VoteRecords) sanitizeHash(h valuehash.Hash) valuehash.Hash {
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

	vrs.voteproof = vrs.vote(blt)

	return vrs.voteproof
}

func (vrs *VoteRecords) vote(blt ballot.Ballot) base.VoteproofV0 {
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

	vrs.set = append(vrs.set, vrs.sanitizeHash(blt.Fact().Hash()).String())

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
	facts := make([]base.Fact, len(vrs.facts))
	var i int
	for k := range vrs.facts {
		facts[i] = vrs.facts[k]
		i++
	}
	voteproof.SetFacts(facts)

	votes := make([]base.VoteproofNodeFact, len(vrs.ballots))

	i = 0
	for k := range vrs.ballots {
		blt := vrs.ballots[k]
		votes[i] = base.NewBaseVoteproofNodeFact(
			blt.Node(),
			vrs.sanitizeHash(blt.Hash()),
			vrs.sanitizeHash(blt.Fact().Hash()),
			blt.FactSignature(),
			blt.Signer(),
		)
		i++
	}
	voteproof.SetVotes(votes)

	return voteproof.Finish()
}
