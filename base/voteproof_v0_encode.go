package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

type VoteproofV0FactUnpacker interface {
	Hash() valuehash.Bytes
	Fact() []byte
}

type VoteproofV0BallotUnpacker interface {
	Hash() valuehash.Bytes
	Address() []byte
}

func (vp *VoteproofV0) unpack( // nolint
	enc encoder.Encoder,
	height Height,
	round Round,
	bSuffrages []AddressDecoder,
	thresholdRatio ThresholdRatio,
	result VoteResultType,
	stage Stage,
	bMajority []byte,
	bFacts [][]byte,
	bVotes [][]byte,
	finishedAt time.Time,
	isClosed bool,
) error {
	if bMajority != nil {
		m, err := DecodeFact(enc, bMajority)
		if err != nil {
			return err
		}
		vp.majority = m
	}

	vp.suffrages = make([]Address, len(bSuffrages))
	for i := range bSuffrages {
		address, err := bSuffrages[i].Encode(enc)
		if err != nil {
			return err
		}
		vp.suffrages[i] = address
	}

	facts := make([]Fact, len(bFacts))
	for i := range bFacts {
		switch fact, err := DecodeFact(enc, bFacts[i]); {
		case err != nil:
			return err
		default:
			facts[i] = fact
		}
	}

	votes := make([]VoteproofNodeFact, len(bVotes))
	for i := range bVotes {
		var nodeFact VoteproofNodeFact
		if err := enc.Decode(bVotes[i], &nodeFact); err != nil {
			return err
		}
		votes[i] = nodeFact
	}

	vp.height = height
	vp.round = round
	vp.thresholdRatio = thresholdRatio
	vp.result = result
	vp.stage = stage
	vp.facts = facts
	vp.votes = votes
	vp.finishedAt = finishedAt
	vp.closed = isClosed

	return nil
}

func (vf *VoteproofNodeFact) unpack(
	enc encoder.Encoder,
	bAddress AddressDecoder,
	blt,
	fact valuehash.Hash,
	factSignature key.Signature,
	bSigner key.PublickeyDecoder,
) error {
	address, err := bAddress.Encode(enc)
	if err != nil {
		return err
	}

	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	vf.address = address
	vf.ballot = blt
	vf.fact = fact
	vf.factSignature = factSignature
	vf.signer = signer

	return nil
}
