package base

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (vp *VoteproofV0) unpack( // nolint
	enc encoder.Encoder,
	height Height,
	round Round,
	threshold Threshold,
	result VoteResultType,
	stage Stage,
	bMajority []byte,
	bFacts,
	bBallots,
	bVotes [][2][]byte,
	finishedAt time.Time,
	isClosed bool,
) error {
	var err error
	var majority Fact
	if bMajority != nil {
		if majority, err = DecodeFact(enc, bMajority); err != nil {
			return err
		}
	}

	facts := map[valuehash.Hash]Fact{}
	for i := range bFacts {
		l := bFacts[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of facts; not [2]bson.Raw")
		}

		var factHash valuehash.Hash
		if factHash, err = valuehash.Decode(enc, l[0]); err != nil {
			return err
		}

		var fact Fact
		if fact, err = DecodeFact(enc, l[1]); err != nil {
			return err
		}

		facts[factHash] = fact
	}

	ballots := map[Address]valuehash.Hash{}
	for i := range bBallots {
		l := bBallots[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of ballots; not [2]bson.Raw")
		}

		var address Address
		if address, err = DecodeAddress(enc, l[0]); err != nil {
			return err
		}

		var ballot valuehash.Hash
		if ballot, err = valuehash.Decode(enc, l[1]); err != nil {
			return err
		}

		ballots[address] = ballot
	}

	votes := map[Address]VoteproofNodeFact{}
	for i := range bVotes {
		l := bVotes[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of votes; not [2]bson.Raw")
		}

		var address Address
		if address, err = DecodeAddress(enc, l[0]); err != nil {
			return err
		}

		var nodeFact VoteproofNodeFact
		if err = enc.Decode(l[1], &nodeFact); err != nil {
			return err
		}

		votes[address] = nodeFact
	}

	vp.height = height
	vp.round = round
	vp.threshold = threshold
	vp.result = result
	vp.stage = stage
	vp.majority = majority
	vp.facts = facts
	vp.ballots = ballots
	vp.votes = votes
	vp.finishedAt = finishedAt
	vp.closed = isClosed

	return nil
}

func (vf *VoteproofNodeFact) unpack(
	enc encoder.Encoder,
	bAddress []byte,
	bFact []byte,
	factSignature key.Signature,
	bSigner []byte,
) error {
	var address Address
	if h, err := DecodeAddress(enc, bAddress); err != nil {
		return err
	} else {
		address = h
	}

	var fact valuehash.Hash
	if h, err := valuehash.Decode(enc, bFact); err != nil {
		return err
	} else {
		fact = h
	}

	var signer key.Publickey
	if h, err := key.DecodePublickey(enc, bSigner); err != nil {
		return err
	} else {
		signer = h
	}

	vf.address = address
	vf.fact = fact
	vf.factSignature = factSignature
	vf.signer = signer

	return nil
}
