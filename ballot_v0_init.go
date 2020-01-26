package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var INITBallotV0Hint hint.Hint = hint.MustHint(INITBallotType, "0.1")

type INITBallotV0 struct {
	BaseBallotV0
	h  valuehash.Hash
	bh valuehash.Hash

	//-x-------------------- hashing parts
	previousBlock valuehash.Hash
	previousRound Round
	voteResult    VoteResult
	//--------------------x-
}

func (ib INITBallotV0) Hint() hint.Hint {
	return INITBallotV0Hint
}

func (ib INITBallotV0) Stage() Stage {
	return StageINIT
}

func (ib INITBallotV0) Hash() valuehash.Hash {
	return ib.h
}

func (ib INITBallotV0) BodyHash() valuehash.Hash {
	return ib.bh
}

func (ib INITBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{ib.BaseBallotV0, ib.previousBlock}, b); err != nil {
		return err
	}

	return nil
}

func (ib INITBallotV0) PreviousBlock() valuehash.Hash {
	return ib.previousBlock
}

func (ib INITBallotV0) PreviousRound() Round {
	return ib.previousRound
}

func (ib INITBallotV0) VoteResult() VoteResult {
	return ib.voteResult
}

func (ib INITBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ib.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ib.BaseBallotV0.Bytes(),
		ib.previousBlock.Bytes(),
		ib.previousRound.Bytes(),
		ib.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ib.BaseBallotV0.IsReadyToSign(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ib.BaseBallotV0.BodyBytes(),
		ib.previousBlock.Bytes(),
		ib.previousRound.Bytes(),
		ib.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib *INITBallotV0) Sign(pk key.Privatekey, b []byte) error {
	var bodyHash valuehash.Hash
	if h, err := ib.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	sig, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b}))
	if err != nil {
		return err
	}
	ib.BaseBallotV0.signer = pk.Publickey()
	ib.BaseBallotV0.signature = sig
	ib.BaseBallotV0.signedAt = localtime.Now()
	ib.bh = bodyHash

	if h, err := ib.GenerateHash(b); err != nil {
		return err
	} else {
		ib.h = h
	}

	return nil
}
