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
	vrs        *sync.Map
	localState *LocalState
}

func NewBallotbox(localState *LocalState) *Ballotbox {
	return &Ballotbox{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballotbox")
		}),
		vrs:        &sync.Map{},
		localState: localState,
	}
}

// Vote receives Ballot and returns VoteRecords, which has VoteRecords.Result()
// and VoteRecords.Majority().
func (bb *Ballotbox) Vote(ballot Ballot) (VoteProof, error) {
	vrs := bb.loadVoteRecords(ballot, true)

	// TODO if VoteRecords is finished, clean up the vrs;
	// - not next height or round
	return vrs.Vote(ballot), nil
}

func (bb *Ballotbox) loadVoteRecords(ballot Ballot, ifNotCreate bool) *VoteRecords {
	bb.Lock()
	defer bb.Unlock()

	key := bb.vrsKey(ballot)

	var vrs *VoteRecords
	if i, found := bb.vrs.Load(key); found {
		vrs = i.(*VoteRecords)
	} else if ifNotCreate {
		vrs = NewVoteRecords(ballot, bb.localState.Policy().Threshold())
		bb.vrs.Store(key, vrs)
	}

	return vrs
}

func (bb *Ballotbox) vrsKey(ballot Ballot) string {
	return fmt.Sprintf("%d-%d-%d", ballot.Height(), ballot.Round(), ballot.Stage())
}
