package isaac

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type Ballotbox struct {
	sync.RWMutex
	*common.ZLogger
	voted     map[string]*Records
	threshold *Threshold
}

func NewBallotbox(threshold *Threshold) *Ballotbox {
	return &Ballotbox{
		ZLogger: common.NewZLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballotbox")
		}),
		voted:     map[string]*Records{},
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

	bb.Lock()
	rs, found := bb.voted[key]
	if !found {
		rs = NewRecords(height, round, stage)
		bb.voted[key] = rs
	}
	bb.Unlock()

	if err := rs.Vote(n, block, lastBlock, lastRound, proposal); err != nil {
		return VoteResult{}, err
	}

	total, threshold := bb.threshold.Get(rs.stage)
	vr := rs.CheckMajority(total, threshold)

	return vr, nil
}

type Records struct {
	sync.RWMutex
	height Height
	round  Round
	stage  Stage
	voted  map[string]map[node.Address]Record
	closed bool
	result VoteResult
}

func NewRecords(height Height, round Round, stage Stage) *Records {
	return &Records{
		height: height,
		round:  round,
		stage:  stage,
		voted:  map[string]map[node.Address]Record{},
	}
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

	key := fmt.Sprintf(
		"%s-%s-%s-%s",
		block.String(),
		lastBlock.String(),
		lastRound.String(),
		proposal.String(),
	)

	nr, found := rs.voted[key]
	if !found {
		nr = map[node.Address]Record{}
		rs.voted[key] = nr
	}

	nr[n] = NewRecord(
		n,
		block,
		lastBlock,
		lastRound,
		proposal,
	)

	return nil
}

func (rs *Records) CheckMajority(total, threshold uint) VoteResult {
	var records []Record

	if rs.IsClosed() {
		rs.RLock()
		defer rs.RUnlock()

		for _, nr := range rs.voted {
			for _, r := range nr {
				records = append(records, r)
			}
		}
		return rs.result.SetRecords(records).SetClosed()
	}

	rs.Lock()
	defer rs.Unlock()

	var keys []string
	var sets []uint
	for k, nr := range rs.voted {
		// TODO filter the old Record
		keys = append(keys, k)
		sets = append(sets, uint(len(nr)))
		for _, r := range nr {
			records = append(records, r)
		}
	}

	vr := NewVoteResult(rs.height, rs.round, rs.stage).
		SetRecords(records)

	idx := common.CheckMajority(total, threshold, sets...)
	switch idx {
	case -1:
		vr = vr.SetAgreement(NotYet)
	case -2:
		vr = vr.SetAgreement(Draw)
		rs.closed = true
	default:
		vr = vr.SetAgreement(Majority)
		for _, r := range rs.voted[keys[idx]] {
			vr = vr.SetBlock(r.block).
				SetLastBlock(r.lastBlock).
				SetLastRound(r.lastRound).
				SetProposal(r.proposal)
			break
		}

		rs.closed = true
	}

	if rs.closed {
		rs.result = vr
	}

	return vr
}

func (rs *Records) IsClosed() bool {
	rs.RLock()
	defer rs.RUnlock()

	return rs.closed
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

func (rc Record) String() string {
	b, _ := json.Marshal(rc) // nolint
	return string(b)
}
