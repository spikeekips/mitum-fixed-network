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

var INITBallotV0Hint hint.Hint = hint.MustHint(INITBallotType, "0.1")

// TODO rename to INITBallotFactV0
type INITBallotFactV0 struct {
	BaseBallotFactV0
	previousBlock valuehash.Hash
	previousRound Round
}

func (ibf INITBallotFactV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ibf.BaseBallotFactV0,
		ibf.previousBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (ibf INITBallotFactV0) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
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
	voteProof     VoteProof
	factHash      valuehash.Hash
	factSignature key.Signature
}

// TODO round argument should be removed, round is already set by
// - VoteProof.Round() + 1: if VoteProof.Stage() == StageINIT) or
// - Round(0): if VoteProof.Stage() == StageACCEPT
func NewINITBallotV0FromLocalState(localState *LocalState, round Round, b []byte) (INITBallotV0, error) {
	lastBlock := localState.LastBlock()
	if lastBlock == nil {
		return INITBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: lastBlock.Height() + 1,
				round:  round,
			},
			previousBlock: lastBlock.Hash(),
			previousRound: lastBlock.Round(),
		},
	}

	var voteProof VoteProof
	if round == 0 {
		voteProof = localState.LastACCEPTVoteProof()
	} else {
		voteProof = localState.LastINITVoteProof()
	}
	ib.voteProof = voteProof

	// TODO NetworkID must be given.
	if err := ib.Sign(localState.Node().Privatekey(), b); err != nil {
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
	if err := isvalid.Check([]isvalid.IsValider{
		ib.BaseBallotV0,
		ib.INITBallotFactV0,
	}, b); err != nil {
		return err
	}

	// TODO validate VoteProof

	return nil
}

func (ib INITBallotV0) VoteProof() VoteProof {
	return ib.voteProof
}

func (ib INITBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ib.IsValid(b); err != nil {
		return nil, err
	}

	var vpb []byte
	if ib.voteProof != nil {
		vpb = ib.voteProof.Bytes()
	}

	e := util.ConcatSlice([][]byte{
		ib.BaseBallotV0.Bytes(),
		ib.INITBallotFactV0.Bytes(),
		ib.bodyHash.Bytes(),
		vpb,
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ib.INITBallotFactV0.IsValid(b); err != nil {
		return nil, err
	}

	var vpb []byte
	if ib.voteProof != nil {
		vpb = ib.voteProof.Bytes()
	}

	e := util.ConcatSlice([][]byte{
		ib.INITBallotFactV0.Bytes(),
		vpb,
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
