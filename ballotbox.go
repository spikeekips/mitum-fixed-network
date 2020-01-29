package mitum

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
)

// Ballotbox collects ballots and keeps track of majority.
type Ballotbox struct {
	sync.RWMutex
	*logging.Logger
	vrs       *sync.Map
	threshold Threshold
}

func NewBallotbox(threshold Threshold) *Ballotbox {
	return &Ballotbox{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballotbox")
		}),
		vrs:       &sync.Map{},
		threshold: threshold,
	}
}

// Vote receives Ballot and returns VoteResult, which has VoteResult.Result()
// and VoteResult.Majority().
func (bb *Ballotbox) Vote(ballot Ballot) (*VoteResult, error) {
	vr := bb.loadVoteResult(ballot, true)

	// TODO if VoteResult is finished, clean up the vrs;
	// - not next height or round
	_, _ = vr.Vote(ballot)

	return vr, nil
}

func (bb *Ballotbox) loadVoteResult(ballot Ballot, ifNotCreate bool) *VoteResult {
	key := bb.vrsKey(ballot)

	var vr *VoteResult
	if i, found := bb.vrs.Load(key); found {
		vr = i.(*VoteResult)
	} else if ifNotCreate {
		vr = NewVoteResult(ballot, bb.threshold)
		bb.vrs.Store(key, vr)
	}

	return vr
}

func (bb *Ballotbox) vrsKey(ballot Ballot) string {
	return fmt.Sprintf("%d-%d-%d", ballot.Height(), ballot.Round(), ballot.Stage())
}
