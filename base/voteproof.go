package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Voteproof interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	logging.LogHintedMarshaler
	ID() string // ID is only unique in local machine
	IsFinished() bool
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteResultType
	Majority() Fact
	Facts() []Fact
	Votes() []VoteproofNodeFact
	ThresholdRatio() ThresholdRatio
	Suffrages() []Address
}

type VoteproofNodeFact struct {
	address       Address
	ballot        valuehash.Hash
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func NewVoteproofNodeFact(
	address Address,
	blt valuehash.Hash,
	fact valuehash.Hash,
	factSignature key.Signature,
	signer key.Publickey,
) VoteproofNodeFact {
	return VoteproofNodeFact{
		address:       address,
		ballot:        blt,
		fact:          fact,
		factSignature: factSignature,
		signer:        signer,
	}
}

func (vf VoteproofNodeFact) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.address,
		vf.ballot,
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, nil, false); err != nil {
		return err
	}

	return vf.signer.Verify(util.ConcatBytesSlice(vf.fact.Bytes(), networkID), vf.factSignature)
}

func (vf VoteproofNodeFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		vf.address.Bytes(),
		vf.ballot.Bytes(),
		vf.fact.Bytes(),
		vf.factSignature.Bytes(),
		vf.signer.Bytes(),
	)
}

func (vf VoteproofNodeFact) Ballot() valuehash.Hash {
	return vf.ballot
}

func (vf VoteproofNodeFact) Fact() valuehash.Hash {
	return vf.fact
}

func (vf VoteproofNodeFact) Signature() key.Signature {
	return vf.factSignature
}

func (vf VoteproofNodeFact) Node() Address {
	return vf.address
}

func (vf VoteproofNodeFact) Signer() key.Publickey {
	return vf.signer
}

type VoteproofCallbacker struct {
	Voteproof
	callback func() error
}

func VoteproofWithCallback(voteproof Voteproof, callback func() error) VoteproofCallbacker {
	return VoteproofCallbacker{Voteproof: voteproof, callback: callback}
}

func (vc VoteproofCallbacker) Callback() error {
	return vc.callback()
}

type Voteproofer interface {
	Voteproof() Voteproof
}

func CompareVoteproof(a, b Voteproof) int {
	if a.Height() > b.Height() {
		return 1
	} else if a.Height() < b.Height() {
		return -1
	}

	if a.Round() > b.Round() {
		return 1
	} else if a.Round() < b.Round() {
		return -1
	}

	if a.Stage() > b.Stage() {
		return 1
	} else if a.Stage() < b.Stage() {
		return -1
	}

	return 0
}
