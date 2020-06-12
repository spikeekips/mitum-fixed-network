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
	Votes() map[Address]VoteproofNodeFact
	ThresholdRatio() ThresholdRatio
	Suffrages() []Address
}

type VoteproofNodeFact struct {
	address       Address
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func NewVoteproofNodeFact(
	address Address,
	fact valuehash.Hash,
	factSignature key.Signature,
	signer key.Publickey,
) VoteproofNodeFact {
	return VoteproofNodeFact{
		address:       address,
		fact:          fact,
		factSignature: factSignature,
		signer:        signer,
	}
}

func (vf VoteproofNodeFact) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.address,
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
		vf.fact.Bytes(),
		vf.factSignature.Bytes(),
		[]byte(vf.signer.String()),
	)
}

func (vf VoteproofNodeFact) Node() Address {
	return vf.address
}

func (vf VoteproofNodeFact) Signer() key.Publickey {
	return vf.signer
}
