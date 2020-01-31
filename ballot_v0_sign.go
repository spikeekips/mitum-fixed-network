package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var SIGNBallotV0Hint hint.Hint = hint.MustHint(INITBallotType, "0.1")

type SIGNBallotV0Fact struct {
	BaseBallotV0Fact
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (sbf SIGNBallotV0Fact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sbf.BaseBallotV0Fact,
		sbf.proposal,
		sbf.newBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (sbf SIGNBallotV0Fact) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{sbf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (sbf SIGNBallotV0Fact) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		sbf.BaseBallotV0Fact.Bytes(),
		sbf.proposal.Bytes(),
		sbf.newBlock.Bytes(),
	})
}

func (sbf SIGNBallotV0Fact) Proposal() valuehash.Hash {
	return sbf.proposal
}

func (sbf SIGNBallotV0Fact) NewBlock() valuehash.Hash {
	return sbf.newBlock
}

type SIGNBallotV0 struct {
	BaseBallotV0
	SIGNBallotV0Fact
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	factHash      valuehash.Hash
	factSignature key.Signature
}

func (sb SIGNBallotV0) Hint() hint.Hint {
	return SIGNBallotV0Hint
}

func (sb SIGNBallotV0) Stage() Stage {
	return StageSIGN
}

func (sb SIGNBallotV0) Hash() valuehash.Hash {
	return sb.h
}

func (sb SIGNBallotV0) BodyHash() valuehash.Hash {
	return sb.bodyHash
}

func (sb SIGNBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sb.BaseBallotV0,
		sb.SIGNBallotV0Fact,
	}, b); err != nil {
		return err
	}

	return nil
}

func (sb SIGNBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := sb.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		sb.BaseBallotV0.Bytes(),
		sb.SIGNBallotV0Fact.Bytes(),
		sb.bodyHash.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb SIGNBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := sb.SIGNBallotV0Fact.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		sb.SIGNBallotV0Fact.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb SIGNBallotV0) Fact() Fact {
	return sb.SIGNBallotV0Fact
}

func (sb SIGNBallotV0) FactHash() valuehash.Hash {
	return sb.factHash
}

func (sb SIGNBallotV0) FactSignature() key.Signature {
	return sb.factSignature
}

func (sb *SIGNBallotV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := sb.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := sb.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	var sig key.Signature
	if s, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b})); err != nil {
		return err
	} else {
		sig = s
	}

	var factHash valuehash.Hash
	if h, err := sb.SIGNBallotV0Fact.Hash(b); err != nil {
		return err
	} else {
		factHash = h
	}

	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	sb.BaseBallotV0.signer = pk.Publickey()
	sb.BaseBallotV0.signature = sig
	sb.BaseBallotV0.signedAt = localtime.Now()
	sb.bodyHash = bodyHash
	sb.factHash = factHash
	sb.factSignature = factSig

	if h, err := sb.GenerateHash(b); err != nil {
		return err
	} else {
		sb.h = h
	}

	return nil
}
