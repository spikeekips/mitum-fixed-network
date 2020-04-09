package base

import (
	"time"

	"github.com/spikeekips/mitum/base/valuehash"
)

func NewTestVoteproofV0(
	height Height,
	round Round,
	threshold Threshold,
	result VoteResultType,
	closed bool,
	stage Stage,
	majority Fact,
	facts map[valuehash.Hash]Fact,
	ballots map[Address]valuehash.Hash,
	votes map[Address]VoteproofNodeFact,
	finishedAt time.Time,
) VoteproofV0 {
	return VoteproofV0{
		height:     height,
		round:      round,
		threshold:  threshold,
		result:     result,
		closed:     closed,
		stage:      stage,
		majority:   majority,
		facts:      facts,
		ballots:    ballots,
		votes:      votes,
		finishedAt: finishedAt,
	}
}
