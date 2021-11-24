package isaac

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/logging"
)

// Ballotbox collects ballots and keeps track of majority.
type Ballotbox struct {
	sync.RWMutex
	*logging.Logging
	vrs           *sync.Map
	suffragesFunc func() []base.Address
	thresholdFunc func() base.Threshold
	latestBallot  base.Ballot
}

func NewBallotbox(suffragesFunc func() []base.Address, thresholdFunc func() base.Threshold) *Ballotbox {
	return &Ballotbox{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballotbox")
		}),
		vrs:           &sync.Map{},
		suffragesFunc: suffragesFunc,
		thresholdFunc: thresholdFunc,
	}
}

// Vote receives Ballot and returns VoteRecords, which has VoteRecords.Result()
// and VoteRecords.Majority().
func (bb *Ballotbox) Vote(blt base.Ballot) (base.Voteproof, error) {
	if err := bb.canVote(blt); err != nil {
		return nil, err
	}

	vrs := bb.loadVoteRecords(blt, true)

	return vrs.Vote(blt), nil
}

func (bb *Ballotbox) Clean(height base.Height) error {
	bb.Lock()
	defer bb.Unlock()

	l := bb.Log().With().Int64("height", height.Int64()).Logger()
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
			err = errors.Errorf("invalid formatted key found: key=%q", k)
			return false
		}

		if h > gh {
			return true
		}

		removeKeys = append(removeKeys, k)
		removes = append(removes, v)

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

func (bb *Ballotbox) LatestBallot() base.Ballot {
	bb.RLock()
	defer bb.RUnlock()

	return bb.latestBallot
}

func (bb *Ballotbox) loadVoteRecords(blt base.Ballot, ifNotCreate bool) *VoteRecords {
	bb.Lock()
	defer bb.Unlock()

	fact := blt.RawFact()

	var lfact base.BallotFact
	if bb.latestBallot != nil {
		lfact = bb.latestBallot.RawFact()
	}

	switch {
	case lfact == nil:
		bb.latestBallot = blt
	case fact.Height() > lfact.Height():
		bb.latestBallot = blt
	case fact.Height() == lfact.Height() && fact.Round() > lfact.Round():
		bb.latestBallot = blt
	case fact.Height() == lfact.Height() && fact.Round() == lfact.Round() && fact.Stage().CanVote():
		if fact.Stage() > bb.latestBallot.RawFact().Stage() {
			bb.latestBallot = blt
		}
	}

	key := bb.vrsKey(fact)

	var vrs *VoteRecords
	if i, found := bb.vrs.Load(key); found {
		vrs = i.(*VoteRecords)
	} else if ifNotCreate {
		vrs = NewVoteRecords(fact.Height(), fact.Round(), fact.Stage(), bb.suffragesFunc(), bb.thresholdFunc())
		bb.vrs.Store(key, vrs)
	}

	return vrs
}

func (*Ballotbox) vrsKey(blt base.BallotFact) string {
	return fmt.Sprintf("%d-%d-%d", blt.Height(), blt.Round(), blt.Stage())
}

func (bb *Ballotbox) canVote(blt base.Ballot) error {
	if !blt.RawFact().Stage().CanVote() {
		return errors.Errorf("this ballot is not for voting; stage=%s", blt.RawFact().Stage())
	}

	var found bool
	for _, a := range bb.suffragesFunc() {
		if a.Equal(blt.FactSign().Node()) {
			found = true
			break
		}
	}

	if !found {
		return errors.Errorf("this ballot is not in suffrages")
	}

	return nil
}
