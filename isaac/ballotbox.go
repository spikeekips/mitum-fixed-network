package isaac

import (
	"fmt"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/logging"
)

// Ballotbox collects ballots and keeps track of majority.
type Ballotbox struct {
	sync.RWMutex
	*logging.Logging
	vrs           *sync.Map
	thresholdFunc func() base.Threshold
}

func NewBallotbox(thresholdFunc func() base.Threshold) *Ballotbox {
	return &Ballotbox{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ballotbox")
		}),
		vrs:           &sync.Map{},
		thresholdFunc: thresholdFunc,
	}
}

// Vote receives Ballot and returns VoteRecords, which has VoteRecords.Result()
// and VoteRecords.Majority().
func (bb *Ballotbox) Vote(blt ballot.Ballot) (base.Voteproof, error) {
	if !blt.Stage().CanVote() {
		return nil, xerrors.Errorf("this ballot is not for voting; stage=%s", blt.Stage())
	}

	vrs := bb.loadVoteRecords(blt, true)

	voteproof := vrs.Vote(blt)

	if voteproof.IsFinished() && !voteproof.IsClosed() {
		// TODO Cleaning VoteRecords may take too long time.
		if err := bb.clean(voteproof.Height(), voteproof.Round()); err != nil {
			return nil, err
		}
	}

	return voteproof, nil
}

func (bb *Ballotbox) loadVoteRecords(blt ballot.Ballot, ifNotCreate bool) *VoteRecords {
	bb.Lock()
	defer bb.Unlock()

	key := bb.vrsKey(blt)

	var vrs *VoteRecords
	if i, found := bb.vrs.Load(key); found {
		vrs = i.(*VoteRecords)
	} else if ifNotCreate {
		vrs = NewVoteRecords(blt, bb.thresholdFunc())
		bb.vrs.Store(key, vrs)
	}

	return vrs
}

func (bb *Ballotbox) clean(height base.Height, round base.Round) error {
	gh := height.Int64()
	gr := round.Uint64()

	var err error
	var removes []interface{}
	bb.vrs.Range(func(k, v interface{}) bool {
		var h int64
		var r uint64
		var s uint8

		var n int
		n, err = fmt.Sscanf(k.(string), "%d-%d-%d", &h, &r, &s)
		if err != nil {
			return false
		}
		if n != 3 {
			err = xerrors.Errorf("invalid formatted key found: key=%q", k)
			return false
		}

		if h != gh {
			removes = append(removes, k)
		}
		if r != gr {
			removes = append(removes, k)
		}

		return true
	})

	if err != nil {
		return err
	}

	if len(removes) < 1 {
		return nil
	}
	for _, k := range removes {
		bb.vrs.Delete(k)
	}

	return nil
}

func (bb *Ballotbox) vrsKey(blt ballot.Ballot) string {
	return fmt.Sprintf("%d-%d-%d", blt.Height(), blt.Round(), blt.Stage())
}
