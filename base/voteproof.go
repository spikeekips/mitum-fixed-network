package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
)

type Voteproof interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	logging.LogHintedMarshaler
	IsFinished() bool
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteResultType
	Majority() Fact
	Ballots() map[Address]valuehash.Hash
	Threshold() Threshold
}

type VoteproofNodeFact struct {
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func NewVoteproofNodeFact(fact valuehash.Hash, factSignature key.Signature, signer key.Publickey) VoteproofNodeFact {
	return VoteproofNodeFact{
		fact:          fact,
		factSignature: factSignature,
		signer:        signer,
	}
}

func (vf VoteproofNodeFact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, nil, false); err != nil {
		return err
	}

	return vf.signer.Verify(util.ConcatBytesSlice(vf.fact.Bytes(), b), vf.factSignature)
}
