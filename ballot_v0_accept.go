package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var ACCEPTBallotV0Hint hint.Hint = hint.MustHint(ACCEPTBallotType, "0.1")

type ACCEPTBallotV0Fact struct {
	BaseBallotV0Fact
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (abf ACCEPTBallotV0Fact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		abf.BaseBallotV0Fact,
		abf.proposal,
		abf.newBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (abf ACCEPTBallotV0Fact) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{abf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (abf ACCEPTBallotV0Fact) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		abf.BaseBallotV0Fact.Bytes(),
		abf.proposal.Bytes(),
		abf.newBlock.Bytes(),
	})
}

func (abf ACCEPTBallotV0Fact) Proposal() valuehash.Hash {
	return abf.proposal
}

func (abf ACCEPTBallotV0Fact) NewBlock() valuehash.Hash {
	return abf.newBlock
}

type ACCEPTBallotV0 struct {
	BaseBallotV0
	ACCEPTBallotV0Fact
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	voteResult    VoteResult
	factHash      valuehash.Hash
	factSignature key.Signature
}

func (ab ACCEPTBallotV0) Hint() hint.Hint {
	return ACCEPTBallotV0Hint
}

func (ab ACCEPTBallotV0) Stage() Stage {
	return StageACCEPT
}

func (ab ACCEPTBallotV0) Hash() valuehash.Hash {
	return ab.h
}

func (ab ACCEPTBallotV0) BodyHash() valuehash.Hash {
	return ab.bodyHash
}

func (ab ACCEPTBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ab.BaseBallotV0,
		ab.ACCEPTBallotV0Fact,
	}, b); err != nil {
		return err
	}

	return nil
}

func (ab ACCEPTBallotV0) VoteResult() VoteResult {
	return ab.voteResult
}

func (ab ACCEPTBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ab.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ab.BaseBallotV0.Bytes(),
		ab.ACCEPTBallotV0Fact.Bytes(),
		ab.bodyHash.Bytes(),
		ab.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ab.ACCEPTBallotV0Fact.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ab.ACCEPTBallotV0Fact.Bytes(),
		ab.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) Fact() Fact {
	return ab.ACCEPTBallotV0Fact
}

func (ab ACCEPTBallotV0) FactHash() valuehash.Hash {
	return ab.factHash
}

func (ab ACCEPTBallotV0) FactSignature() key.Signature {
	return ab.factSignature
}

func (ab *ACCEPTBallotV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := ab.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := ab.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	sig, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b}))
	if err != nil {
		return err
	}

	var factHash valuehash.Hash
	if h, err := ab.ACCEPTBallotV0Fact.Hash(b); err != nil {
		return err
	} else {
		factHash = h
	}

	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	ab.BaseBallotV0.signer = pk.Publickey()
	ab.BaseBallotV0.signature = sig
	ab.BaseBallotV0.signedAt = localtime.Now()
	ab.bodyHash = bodyHash
	ab.factHash = factHash
	ab.factSignature = factSig

	if h, err := ab.GenerateHash(b); err != nil {
		return err
	} else {
		ab.h = h
	}

	return nil
}
