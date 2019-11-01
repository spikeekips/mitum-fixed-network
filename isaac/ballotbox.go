package isaac

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type Ballotbox struct {
	*common.Logger
	voted     *sync.Map
	threshold *Threshold
}

func NewBallotbox(threshold *Threshold) *Ballotbox {
	return &Ballotbox{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballotbox")
		}),
		voted:     &sync.Map{},
		threshold: threshold,
	}
}

func (bb *Ballotbox) Vote(
	n node.Address,
	height Height,
	round Round,
	stage Stage,
	block hash.Hash,
	lastBlock hash.Hash,
	lastRound Round,
	proposal hash.Hash,
) (VoteResult, error) {
	key := fmt.Sprintf(
		"%v-%v-%v",
		height.String(),
		round,
		stage.String(),
	)

	var rs *Records
	if i, found := bb.voted.Load(key); !found {
		rs = NewRecords(height, round, stage)
		_ = rs.SetLogger(*bb.Log())
		bb.voted.Store(key, rs)
	} else {
		rs = i.(*Records)
	}

	if err := rs.Vote(n, block, lastBlock, lastRound, proposal); err != nil {
		return VoteResult{}, err
	}

	total, threshold := bb.threshold.Get(rs.stage)
	vr := rs.CheckMajority(total, threshold)

	return vr, nil
}

func (bb *Ballotbox) Tidy(height Height, round Round) {
	var keys []string
	prefix := fmt.Sprintf("%v-", height.String())
	bb.voted.Range(func(k, v interface{}) bool {
		key := k.(string)
		if !strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
			return true
		}

		var rs *Records
		i, found := bb.voted.Load(key)
		if !found {
			keys = append(keys, key)
			return true
		}

		rs = i.(*Records)
		if rs.round < round {
			keys = append(keys, key)
		}

		return true
	})

	for _, key := range keys {
		bb.voted.Delete(key)
	}
	bb.Log().Debug().
		Strs("records", keys).
		Msg("tidy the vote records")
}

type Records struct {
	sync.RWMutex
	*common.Logger
	height Height
	round  Round
	stage  Stage
	voted  *sync.Map
	result VoteResult
}

func NewRecords(height Height, round Round, stage Stage) *Records {
	return &Records{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "records")
		}),
		height: height,
		round:  round,
		stage:  stage,
		voted:  &sync.Map{},
	}
}

func (rs *Records) key(
	block hash.Hash,
	lastBlock hash.Hash,
	lastRound Round,
	proposal hash.Hash,
) string {
	return fmt.Sprintf(
		"%s-%s-%s-%s",
		block.String(),
		lastBlock.String(),
		lastRound.String(),
		proposal.String(),
	)
}

func (rs *Records) Vote(
	n node.Address,
	block hash.Hash,
	lastBlock hash.Hash,
	lastRound Round,
	proposal hash.Hash,
) error {
	rs.Lock()
	defer rs.Unlock()

	key := rs.key(block, lastBlock, lastRound, proposal)

	var nr *NodesRecord
	if i, found := rs.voted.Load(key); !found {
		nr = NewNodesRecord()
		rs.voted.Store(key, nr)
	} else {
		nr = i.(*NodesRecord)
	}

	_ = nr.Vote(n, block, lastBlock, lastRound, proposal)

	return nil
}

func (rs *Records) CheckMajority(total, threshold uint) VoteResult {
	rs.Lock()
	defer rs.Unlock()

	l := rs.Log().With().
		Str("height", rs.height.String()).
		Uint64("round", rs.round.Uint64()).
		Uint("total", total).
		Uint("threshold", threshold).
		Str("stage", rs.stage.String()).
		Bool("is_finished", rs.result.IsFinished()).
		Bool("is_closed", rs.result.IsClosed()).
		Logger()

	if rs.result.IsFinished() {
		l.Debug().Msg("check majority, but closed")

		return rs.result.SetRecords(rs.Records()).SetClosed()
	}

	var records []Record
	var keys []string
	var sets []uint

	rs.voted.Range(func(k, v interface{}) bool {
		nrs := v.(*NodesRecord).Records()

		keys = append(keys, k.(string))
		sets = append(sets, uint(len(nrs)))
		records = append(records, nrs...)

		return true
	})

	vr := NewVoteResult(rs.height, rs.round, rs.stage).
		SetRecords(records)

	idx := common.CheckMajority(total, threshold, sets...)
	switch idx {
	case -1:
		vr = vr.SetAgreement(NotYet)
	case -2:
		vr = vr.SetAgreement(Draw)
	default:
		vr = vr.SetAgreement(Majority)

		i, _ := rs.voted.Load(keys[idx])
		for _, r := range i.(*NodesRecord).Records() {
			vr = vr.SetBlock(r.block).
				SetLastBlock(r.lastBlock).
				SetLastRound(r.lastRound).
				SetProposal(r.proposal)
			break
		}

	}

	l.Debug().
		Uints("set", sets).
		Bool("is_finished", vr.IsFinished()).
		Str("agreement", vr.Agreement().String()).
		Msg("check majority")

	if vr.IsFinished() {
		rs.result = vr
	}

	return vr
}

func (rs *Records) IsFinished() bool {
	rs.RLock()
	defer rs.RUnlock()

	return rs.result.IsFinished()
}

func (rs *Records) IsClosed() bool {
	rs.RLock()
	defer rs.RUnlock()

	return rs.result.IsClosed()
}

func (rs *Records) Result() VoteResult {
	rs.RLock()
	defer rs.RUnlock()

	return rs.result
}

func (rs *Records) Records() []Record {
	var records []Record
	rs.voted.Range(func(k, v interface{}) bool {
		records = append(records, v.(*NodesRecord).Records()...)
		return true
	})

	return records
}

type Record struct {
	node      node.Address
	block     hash.Hash
	lastBlock hash.Hash
	lastRound Round
	proposal  hash.Hash
	votedAt   common.Time

	// TODO needs ballot hash
}

func NewRecord(n node.Address, block hash.Hash, lastBlock hash.Hash, lastRound Round, proposal hash.Hash) Record {
	return Record{
		node:      n,
		block:     block,
		lastBlock: lastBlock,
		lastRound: lastRound,
		proposal:  proposal,
		votedAt:   common.Now(),
	}
}

func (rc Record) Node() node.Address {
	return rc.node
}

func (rc Record) Block() hash.Hash {
	return rc.block
}

func (rc Record) LastBlock() hash.Hash {
	return rc.lastBlock
}

func (rc Record) LastRound() Round {
	return rc.lastRound
}

func (rc Record) Proposal() hash.Hash {
	return rc.proposal
}

func (rc Record) VotedAt() common.Time {
	return rc.votedAt
}

func (rc Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"node":       rc.node,
		"block":      rc.block,
		"last_block": rc.lastBlock,
		"last_round": rc.lastRound,
		"proposal":   rc.proposal,
		"voted_at":   rc.votedAt,
	})
}

func (rc Record) MarshalZerologObject(e *zerolog.Event) {
	e.Object("node", rc.node)
	e.Object("block", rc.block)
	e.Object("last_block", rc.lastBlock)
	e.Uint64("last_round", rc.lastRound.Uint64())
	e.Object("proposal", rc.proposal)
	e.Time("voted_at", rc.votedAt.Time)
}

func (rc Record) String() string {
	b, _ := json.Marshal(rc) // nolint
	return string(b)
}

type NodesRecord struct {
	voted *sync.Map
}

func NewNodesRecord() *NodesRecord {
	return &NodesRecord{voted: &sync.Map{}}
}

func (nr *NodesRecord) Vote(
	n node.Address,
	block hash.Hash,
	lastBlock hash.Hash,
	lastRound Round,
	proposal hash.Hash,
) *NodesRecord {
	nr.voted.Store(n, NewRecord(n, block, lastBlock, lastRound, proposal))

	return nr
}

func (nr *NodesRecord) Records() []Record {
	var rs []Record
	nr.voted.Range(func(k, v interface{}) bool {
		rs = append(rs, v.(Record))
		return true
	})

	return rs
}
