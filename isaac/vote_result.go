package isaac

import (
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/hash"
)

type Agreement uint

const (
	NotYet Agreement = iota
	Draw
	Majority
)

func (ag Agreement) MarshalJSON() ([]byte, error) {
	return json.Marshal(ag.String())
}

func (ag Agreement) String() string {
	switch ag {
	case NotYet:
		return "NOTYET"
	case Draw:
		return "DRAW"
	case Majority:
		return "MAJORITY"
	default:
		return ""
	}
}

type VoteResult struct {
	height    Height
	round     Round
	stage     Stage
	proposal  hash.Hash
	block     hash.Hash
	lastBlock hash.Hash
	lastRound Round
	records   []Record
	agreement Agreement
	closed    bool
}

func NewVoteResult(
	height Height,
	round Round,
	stage Stage,
) VoteResult {
	return VoteResult{
		height: height,
		round:  round,
		stage:  stage,
	}
}

func (vr VoteResult) Agreement() Agreement {
	return vr.agreement
}

func (vr VoteResult) GotDraw() bool {
	return vr.agreement == Draw
}

func (vr VoteResult) GotMajority() bool {
	return vr.agreement == Majority
}

func (vr VoteResult) IsFinished() bool {
	switch vr.agreement {
	case Draw, Majority:
		return true
	default:
		return false
	}
}

func (vr VoteResult) Height() Height {
	return vr.height
}

func (vr VoteResult) Round() Round {
	return vr.round
}

func (vr VoteResult) Stage() Stage {
	return vr.stage
}

func (vr VoteResult) Proposal() hash.Hash {
	return vr.proposal
}

func (vr VoteResult) Records() []Record {
	return vr.records
}

func (vr VoteResult) SetProposal(proposal hash.Hash) VoteResult {
	vr.proposal = proposal
	return vr
}

func (vr VoteResult) Block() hash.Hash {
	return vr.block
}

func (vr VoteResult) SetBlock(block hash.Hash) VoteResult {
	vr.block = block
	return vr
}

func (vr VoteResult) LastBlock() hash.Hash {
	return vr.lastBlock
}

func (vr VoteResult) SetLastBlock(lastBlock hash.Hash) VoteResult {
	vr.lastBlock = lastBlock
	return vr
}

func (vr VoteResult) LastRound() Round {
	return vr.lastRound
}

func (vr VoteResult) SetLastRound(lastRound Round) VoteResult {
	vr.lastRound = lastRound
	return vr
}

func (vr VoteResult) SetAgreement(agreement Agreement) VoteResult {
	vr.agreement = agreement
	return vr
}

func (vr VoteResult) SetRecords(records []Record) VoteResult {
	vr.records = records
	return vr
}

func (vr VoteResult) IsClosed() bool {
	return vr.closed
}

func (vr VoteResult) SetClosed() VoteResult {
	vr.closed = true
	return vr
}

func (vr VoteResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"height":     vr.height,
		"round":      vr.round,
		"stage":      vr.stage,
		"proposal":   vr.proposal,
		"records":    vr.records,
		"agreement":  vr.agreement,
		"closed":     vr.closed,
		"last_block": vr.lastBlock,
		"last_round": vr.lastRound,
	})
}

func (vr VoteResult) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64("height", vr.height.Uint64())
	e.Uint64("round", vr.round.Uint64())
	e.Str("stage", vr.stage.String())
	e.Object("proposal", vr.proposal)

	rs := zerolog.Arr()
	for _, r := range vr.records {
		rs.Object(r)
	}
	e.Array("records", rs)
	e.Str("agreement", vr.agreement.String())
	e.Bool("closed", vr.closed)
	e.Object("block", vr.block)
	e.Object("last_block", vr.lastBlock)
	e.Uint64("last_round", vr.lastRound.Uint64())
}

func (vr VoteResult) String() string {
	b, _ := json.Marshal(vr) // nolint
	return string(b)
}
