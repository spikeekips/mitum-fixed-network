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

type SIGNBallotV0 struct {
	BaseBallotV0
	h  valuehash.Hash
	bh valuehash.Hash

	//-x-------------------- hashing parts
	proposal valuehash.Hash
	newBlock valuehash.Hash
	//--------------------x-
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
	return sb.bh
}

func (sb SIGNBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{sb.BaseBallotV0, sb.proposal, sb.newBlock}, b); err != nil {
		return err
	}

	return nil
}

func (sb SIGNBallotV0) Proposal() valuehash.Hash {
	return sb.proposal
}

func (sb SIGNBallotV0) NewBlock() valuehash.Hash {
	return sb.newBlock
}

func (sb SIGNBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := sb.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		sb.BaseBallotV0.Bytes(),
		sb.proposal.Bytes(),
		sb.newBlock.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb SIGNBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := sb.BaseBallotV0.IsReadyToSign(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		sb.BaseBallotV0.BodyBytes(),
		sb.proposal.Bytes(),
		sb.newBlock.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb *SIGNBallotV0) Sign(pk key.Privatekey, b []byte) error {
	var bodyHash valuehash.Hash
	if h, err := sb.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	sig, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b}))
	if err != nil {
		return err
	}
	sb.BaseBallotV0.signer = pk.Publickey()
	sb.BaseBallotV0.signature = sig
	sb.BaseBallotV0.signedAt = localtime.Now()
	sb.bh = bodyHash

	if h, err := sb.GenerateHash(b); err != nil {
		return err
	} else {
		sb.h = h
	}

	return nil
}
