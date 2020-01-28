package mitum

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

type VoteRecord interface {
	Node() Address
	Key() string
}

func NewVoteRecord(ballot Ballot) (VoteRecord, error) {
	votedAt := localtime.Now()
	switch ballot.Stage() {
	case StageINIT:
		ib := ballot.(INITBallot)
		return VoteRecordINIT{
			node:          ballot.Node(),
			previousBlock: ib.PreviousBlock(),
			previousRound: ib.PreviousRound(),
			votedAt:       votedAt,
		}, nil
	case StageSIGN:
		sb := ballot.(SIGNBallot)
		return VoteRecordSIGN{
			node:     ballot.Node(),
			proposal: sb.Proposal(),
			newBlock: sb.NewBlock(),
			votedAt:  votedAt,
		}, nil
	case StageACCEPT:
		ab := ballot.(ACCEPTBallot)
		return VoteRecordACCEPT{
			node:     ballot.Node(),
			proposal: ab.Proposal(),
			newBlock: ab.NewBlock(),
			votedAt:  votedAt,
		}, nil
	}

	return nil, InvalidStageError.Wrapf("unknown stage found from ballot: stage=%d", uint8(ballot.Stage()))
}

func CheckVoteRecordsMajority(threshold Threshold, vrs map[Address]VoteRecord) (VoteResultType, VoteRecord) {
	records := map[string]VoteRecord{}
	collected := map[string]int{}
	for _, vr := range vrs {
		key := vr.Key()

		collected[key]++
		if _, found := records[key]; !found {
			records[key] = vr
		}
	}

	byCount := map[uint]string{}
	sets := make([]uint, len(collected))
	var n int
	for k, c := range collected {
		sets[n] = uint(c)
		byCount[uint(c)] = k
		n++
	}

	if len(sets) > 1 {
		sort.Slice(sets, func(i, j int) bool { return sets[i] > sets[j] })
	}

	c := FindMajority(threshold.Total, threshold.Threshold, sets...)
	switch c {
	case -1:
		return VoteResultNotYet, nil
	case -2:
		return VoteResultDraw, nil
	}

	return VoteResultMajority, records[byCount[sets[c]]]
}

type VoteRecordINIT struct {
	node          Address
	votedAt       time.Time
	previousBlock valuehash.Hash
	previousRound Round
}

func (vrc VoteRecordINIT) Node() Address {
	return vrc.node
}

func (vrc VoteRecordINIT) Key() string {
	return fmt.Sprintf("%x-%d", vrc.previousBlock.Bytes(), vrc.previousRound)
}

type VoteRecordSIGN struct {
	node     Address
	votedAt  time.Time
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (vrc VoteRecordSIGN) Node() Address {
	return vrc.node
}

func (vrc VoteRecordSIGN) Key() string {
	return fmt.Sprintf("%x-%d", vrc.proposal.Bytes(), vrc.newBlock.Bytes())
}

type VoteRecordACCEPT struct {
	node     Address
	votedAt  time.Time
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (vrc VoteRecordACCEPT) Node() Address {
	return vrc.node
}

func (vrc VoteRecordACCEPT) Key() string {
	return fmt.Sprintf("%x-%d", vrc.proposal.Bytes(), vrc.newBlock.Bytes())
}

type VoteRecords struct {
	sync.RWMutex
	height    Height
	round     Round
	stage     Stage
	threshold Threshold
	votes     *sync.Map // key: node, value: VoteRecord
	vr        VoteResult
}

func NewVoteRecords(ballot Ballot, threshold Threshold) *VoteRecords {
	return &VoteRecords{
		height:    ballot.Height(),
		round:     ballot.Round(),
		stage:     ballot.Stage(),
		threshold: threshold,
		votes:     &sync.Map{},
		vr:        NewVoteResult(ballot, threshold),
	}
}

func (vrs *VoteRecords) Height() Height {
	return vrs.height
}

func (vrs *VoteRecords) Round() Round {
	return vrs.round
}

func (vrs *VoteRecords) Stage() Stage {
	return vrs.stage
}

func (vrs *VoteRecords) Bytes() []byte {
	return nil
}

func (vrs *VoteRecords) Result() VoteResult {
	return vrs.vr
}

func (vrs *VoteRecords) VoteRecord(node Address) (VoteRecord, bool) {
	i, found := vrs.votes.Load(node)

	if !found {
		return nil, false
	}

	return i.(VoteRecord), true
}

func (vrs *VoteRecords) VoteCount() int {
	var count int
	vrs.votes.Range(func(k, v interface{}) bool {
		count++
		return true
	})

	return count
}

func (vrs *VoteRecords) Vote(ballot Ballot) (VoteResult, error) {
	vrs.Lock()
	defer vrs.Unlock()

	// NOTE disallow to vote multiple times by same node.
	_, found := vrs.votes.Load(ballot.Node())
	if found {
		return vrs.vr, nil
	}

	vrc, err := NewVoteRecord(ballot)
	if err != nil {
		return vrs.vr, err
	}

	vrs.votes.Store(ballot.Node(), vrc)

	if vrs.Result().IsFinished() {
		return vrs.vr, nil
	} else if vrs.VoteCount() < int(vrs.threshold.Threshold) {
		return vrs.vr, nil
	}

	return vrs.CheckMajority(), nil
}

func (vrs *VoteRecords) CheckMajority() VoteResult {
	votes := map[Address]VoteRecord{}
	vrs.votes.Range(func(k, v interface{}) bool {
		votes[k.(Address)] = v.(VoteRecord)

		return true
	})

	result, majority := CheckVoteRecordsMajority(vrs.threshold, votes)
	if result == VoteResultNotYet {
		return vrs.vr
	}

	vrs.vr.votes = votes
	vrs.vr.result = result
	vrs.vr.majority = majority

	return vrs.vr
}
