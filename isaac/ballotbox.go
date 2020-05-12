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

	return vrs.Vote(blt), nil
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

func (bb *Ballotbox) Clean(height base.Height) error {
	bb.Log().Debug().Hinted("height", height).Msg("trying to clean unused records")

	gh := height.Int64()

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

		if h > gh {
			return true
		}

		removes = append(removes, k)

		bb.Log().Debug().
			Int64("height", h).Uint64("round", r).Str("stage", base.Stage(s).String()).
			Msg("records will be removed")

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
