package base

import (
	"time"

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
	bFacts []byte,
	bVotes []byte,
	finishedAt time.Time,
	isClosed bool,
) error {
	if bMajority != nil {
		m, err := DecodeFact(bMajority, enc)
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

	hfacts, err := enc.DecodeSlice(bFacts)
	if err != nil {
		return err
	}
	facts := make([]Fact, len(hfacts))
	for i := range hfacts {
		j, ok := hfacts[i].(Fact)
		if !ok {
			return util.WrongTypeError.Errorf("expected Fact, not %T", hfacts[i])
		}
		facts[i] = j
	}

	hvotes, err := enc.DecodeSlice(bVotes)
	if err != nil {
		return err
	}
	votes := make([]VoteproofNodeFact, len(hvotes))
	for i := range hvotes {
		j, ok := hvotes[i].(VoteproofNodeFact)
		if !ok {
			return util.WrongTypeError.Errorf("expected VoteproofNodeFact, not %T", hvotes[i])
		}
		votes[i] = j
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
