package isaac

import (
	"encoding/json"

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
	})
}

func (vr VoteResult) String() string {
	b, _ := json.Marshal(vr)
	return string(b)
}
