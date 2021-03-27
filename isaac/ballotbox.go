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
	suffragesFunc func() []base.Address
	thresholdFunc func() base.Threshold
	latestBallot  ballot.Ballot
}

func NewBallotbox(suffragesFunc func() []base.Address, thresholdFunc func() base.Threshold) *Ballotbox {
	return &Ballotbox{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ballotbox")
		}),
		vrs:           &sync.Map{},
		suffragesFunc: suffragesFunc,
		thresholdFunc: thresholdFunc,
	}
}

// Vote receives Ballot and returns VoteRecords, which has VoteRecords.Result()
// and VoteRecords.Majority().
func (bb *Ballotbox) Vote(blt ballot.Ballot) (base.Voteproof, error) {
	if err := bb.canVote(blt); err != nil {
		return nil, err
	}

	vrs := bb.loadVoteRecords(blt, true)

	return vrs.Vote(blt), nil
}

func (bb *Ballotbox) Clean(height base.Height) error {
	bb.Lock()
	defer bb.Unlock()

	l := bb.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("height", height)
	})
	l.Debug().Msg("trying to clean unused records")

	gh := height.Int64()

	var err error
	var removeKeys []interface{}
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

		removeKeys = append(removeKeys, k)
		removes = append(removes, v)

		l.Debug().
			Int64("height", h).Uint64("round", r).Str("stage", base.Stage(s).String()).
			Msg("records will be removed")

		return true
	})

	if err != nil {
		return err
	}

	if len(removeKeys) < 1 {
		return nil
	}
	for i := range removeKeys {
		bb.vrs.Delete(removeKeys[i])
	}

	for i := range removes {
		voteRecordsPoolPut(removes[i].(*VoteRecords))
	}

	l.Debug().Int("records", len(removeKeys)).Msg("records removed")

	return nil
}

func (bb *Ballotbox) LatestBallot() ballot.Ballot {
	bb.RLock()
	defer bb.RUnlock()

	return bb.latestBallot
}

func (bb *Ballotbox) loadVoteRecords(blt ballot.Ballot, ifNotCreate bool) *VoteRecords {
	bb.Lock()
	defer bb.Unlock()

	switch l := bb.latestBallot; {
	case l == nil:
		bb.latestBallot = blt
	case blt.Height() > l.Height():
		bb.latestBallot = blt
	case blt.Height() == l.Height() && blt.Round() > l.Round():
		bb.latestBallot = blt
	case blt.Height() == l.Height() && blt.Round() == l.Round() && blt.Stage().CanVote():
		if blt.Stage() > bb.latestBallot.Stage() {
			if _, ok := blt.(base.Voteproofer); ok {
				bb.latestBallot = blt
			}
		}
	}

	key := bb.vrsKey(blt)

	var vrs *VoteRecords
	if i, found := bb.vrs.Load(key); found {
		vrs = i.(*VoteRecords)
	} else if ifNotCreate {
		vrs = NewVoteRecords(blt, bb.suffragesFunc(), bb.thresholdFunc())
		bb.vrs.Store(key, vrs)
	}

	return vrs
}

func (bb *Ballotbox) vrsKey(blt ballot.Ballot) string {
	return fmt.Sprintf("%d-%d-%d", blt.Height(), blt.Round(), blt.Stage())
}

func (bb *Ballotbox) canVote(blt ballot.Ballot) error {
	if !blt.Stage().CanVote() {
		return xerrors.Errorf("this ballot is not for voting; stage=%s", blt.Stage())
	}

	var found bool
	for _, a := range bb.suffragesFunc() {
		if a.Equal(blt.Node()) {
			found = true
			break
		}
	}

	if !found {
		return xerrors.Errorf("this ballot is not in suffrages")
	}

	return nil
}
