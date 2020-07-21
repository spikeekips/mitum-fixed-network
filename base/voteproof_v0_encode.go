package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
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
	var majority Fact
	if bMajority != nil {
		if m, err := DecodeFact(enc, bMajority); err != nil {
			return err
		} else {
			majority = m
		}
	}

	var suffrages []Address
	for i := range bSuffrages {
		if address, err := bSuffrages[i].Encode(enc); err != nil {
			return err
		} else {
			suffrages = append(suffrages, address)
		}
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
		} else {
			votes[i] = nodeFact
		}
	}

	vp.height = height
	vp.round = round
	vp.suffrages = suffrages
	vp.thresholdRatio = thresholdRatio
	vp.result = result
	vp.stage = stage
	vp.majority = majority
	vp.facts = facts
	vp.votes = votes
	vp.finishedAt = finishedAt
	vp.closed = isClosed
	vp.id = util.UUID().String()

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
	var address Address
	if h, err := bAddress.Encode(enc); err != nil {
		return err
	} else {
		address = h
	}

	var signer key.Publickey
	if k, err := bSigner.Encode(enc); err != nil {
		return err
	} else {
		signer = k
	}

	vf.address = address
	vf.ballot = blt
	vf.fact = fact
	vf.factSignature = factSignature
	vf.signer = signer

	return nil
}
