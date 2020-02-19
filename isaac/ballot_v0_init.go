package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

var (
	INITBallotV0Hint     hint.Hint = hint.MustHint(INITBallotType, "0.1")
	INITBallotFactV0Hint hint.Hint = hint.MustHint(INITBallotFactType, "0.1")
)

type INITBallotFactV0 struct {
	BaseBallotFactV0
	previousBlock valuehash.Hash
	previousRound Round
}

func NewINITBallotFactV0(
	height Height,
	round Round,
	previousBlock valuehash.Hash,
	previousRound Round,
) INITBallotFactV0 {
	return INITBallotFactV0{
		BaseBallotFactV0: NewBaseBallotFactV0(height, round),
		previousBlock:    previousBlock,
		previousRound:    previousRound,
	}
}

func (ibf INITBallotFactV0) Hint() hint.Hint {
	return INITBallotFactV0Hint
}

func (ibf INITBallotFactV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ibf.BaseBallotFactV0,
		ibf.previousBlock,
	}, b, false); err != nil {
		return err
	}

	return nil
}

func (ibf INITBallotFactV0) Hash(b []byte) (valuehash.Hash, error) {
	e := util.ConcatSlice([][]byte{ibf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (ibf INITBallotFactV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		ibf.BaseBallotFactV0.Bytes(),
		ibf.previousBlock.Bytes(),
		ibf.previousRound.Bytes(),
	})
}

func (ibf INITBallotFactV0) PreviousBlock() valuehash.Hash {
	return ibf.previousBlock
}

func (ibf INITBallotFactV0) PreviousRound() Round {
	return ibf.previousRound
}

type INITBallotV0 struct {
	BaseBallotV0
	INITBallotFactV0
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	voteproof     Voteproof
	factHash      valuehash.Hash
	factSignature key.Signature
}

func NewINITBallotV0FromLocalstate(localstate *Localstate, round Round, b []byte) (INITBallotV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return INITBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localstate.Node().Address(),
		},
		INITBallotFactV0: NewINITBallotFactV0(
			lastBlock.Height()+1,
			round,
			lastBlock.Hash(),
			lastBlock.Round(),
		),
	}

	var voteproof Voteproof
	if round == 0 {
		voteproof = localstate.LastACCEPTVoteproof()
	} else {
		voteproof = localstate.LastINITVoteproof()
	}
	ib.voteproof = voteproof

	// TODO NetworkID must be given.
	if err := ib.Sign(localstate.Node().Privatekey(), b); err != nil {
		return INITBallotV0{}, err
	}

	return ib, nil
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
	return ib.bodyHash
}

func (ib INITBallotV0) IsValid(b []byte) error {
	if ib.voteproof == nil {
		return xerrors.Errorf("empty Voteproof")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ib.BaseBallotV0,
		ib.INITBallotFactV0,
		ib.voteproof,
	}, b, false); err != nil {
		return err
	}

	return nil
}

func (ib INITBallotV0) Voteproof() Voteproof {
	return ib.voteproof
}

func (ib INITBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ib.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ib.BaseBallotV0.Bytes(),
		ib.INITBallotFactV0.Bytes(),
		ib.bodyHash.Bytes(),
		ib.voteproof.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ib.INITBallotFactV0.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ib.INITBallotFactV0.Bytes(),
		ib.voteproof.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) Fact() Fact {
	return ib.INITBallotFactV0
}

func (ib INITBallotV0) FactHash() valuehash.Hash {
	return ib.factHash
}

func (ib INITBallotV0) FactSignature() key.Signature {
	return ib.factSignature
}

func (ib *INITBallotV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := ib.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}
	if err := ib.INITBallotFactV0.IsValid(b); err != nil {
		return err
	}

	// body signature
	var bodyHash valuehash.Hash
	if h, err := ib.GenerateBodyHash(b); err != nil {
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

	// fact signature
	var factHash valuehash.Hash
	if h, err := ib.INITBallotFactV0.Hash(b); err != nil {
		return err
	} else {
		factHash = h
	}

	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	ib.BaseBallotV0.signer = pk.Publickey()
	ib.BaseBallotV0.signature = sig
	ib.BaseBallotV0.signedAt = localtime.Now()
	ib.bodyHash = bodyHash
	ib.factHash = factHash
	ib.factSignature = factSig

	if h, err := ib.GenerateHash(b); err != nil {
		return err
	} else {
		ib.h = h
	}

	return nil
}
