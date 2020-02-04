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

// TODO rename to INITBallotFactV0
type INITBallotV0Fact struct {
	BaseBallotV0Fact
	previousBlock valuehash.Hash
	previousRound Round
}

func (ibf INITBallotV0Fact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ibf.BaseBallotV0Fact,
		ibf.previousBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (ibf INITBallotV0Fact) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{ibf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (ibf INITBallotV0Fact) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		ibf.BaseBallotV0Fact.Bytes(),
		ibf.previousBlock.Bytes(),
		ibf.previousRound.Bytes(),
	})
}

func (ibf INITBallotV0Fact) PreviousBlock() valuehash.Hash {
	return ibf.previousBlock
}

func (ibf INITBallotV0Fact) PreviousRound() Round {
	return ibf.previousRound
}

type INITBallotV0 struct {
	BaseBallotV0
	INITBallotV0Fact
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
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		INITBallotV0Fact: INITBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: localState.LastBlockHeight() + 1,
				round:  round,
			},
			previousBlock: localState.LastBlockHash(),
			previousRound: localState.LastBlockRound(),
		},
	}
	ib.voteProof = localState.LastINITVoteProof()

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
		ib.INITBallotV0Fact,
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

	e := util.ConcatSlice([][]byte{
		ib.BaseBallotV0.Bytes(),
		ib.INITBallotV0Fact.Bytes(),
		ib.bodyHash.Bytes(),
		ib.voteProof.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ib.INITBallotV0Fact.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ib.INITBallotV0Fact.Bytes(),
		ib.voteProof.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ib INITBallotV0) Fact() Fact {
	return ib.INITBallotV0Fact
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
	if err := ib.INITBallotV0Fact.IsValid(b); err != nil {
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
	if h, err := ib.INITBallotV0Fact.Hash(b); err != nil {
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
