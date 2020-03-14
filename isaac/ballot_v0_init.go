package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	INITBallotV0Hint     hint.Hint = hint.MustHint(INITBallotType, "0.0.1")
	INITBallotFactV0Hint hint.Hint = hint.MustHint(INITBallotFactType, "0.0.1")
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

func (ibf INITBallotFactV0) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ibf.BaseBallotFactV0,
		ibf.previousBlock,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (ibf INITBallotFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(ibf.Bytes())
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
	voteproof Voteproof
}

func NewINITBallotV0(
	localstate *Localstate,
	height Height,
	round Round,
	previousBlock valuehash.Hash,
	previousRound Round,
	voteproof Voteproof,
	networkID []byte,
) (INITBallotV0, error) {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localstate.Node().Address(),
		},
		INITBallotFactV0: NewINITBallotFactV0(
			height,
			round,
			previousBlock,
			previousRound,
		),
		voteproof: voteproof,
	}

	if err := ib.Sign(localstate.Node().Privatekey(), networkID); err != nil {
		return INITBallotV0{}, err
	}

	return ib, nil
}

func NewINITBallotV0FromLocalstate(localstate *Localstate, round Round) (INITBallotV0, error) {
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

	if err := ib.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return INITBallotV0{}, err
	}

	return ib, nil
}

func (ib INITBallotV0) Hash() valuehash.Hash {
	return ib.BaseBallotV0.Hash()
}

func (ib INITBallotV0) Hint() hint.Hint {
	return INITBallotV0Hint
}

func (ib INITBallotV0) Stage() Stage {
	return StageINIT
}

func (ib INITBallotV0) IsValid(b []byte) error {
	if ib.Height() == Height(0) {
		if ib.voteproof != nil {
			return xerrors.Errorf("not empty Voteproof for genesis INITBallot")
		}

		if err := isvalid.Check([]isvalid.IsValider{
			ib.BaseBallotV0,
			ib.INITBallotFactV0,
		}, b, false); err != nil {
			return err
		}
	} else {
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
	}

	if err := IsValidBallot(ib, b); err != nil {
		return err
	}

	return nil
}

func (ib INITBallotV0) Voteproof() Voteproof {
	return ib.voteproof
}

func (ib INITBallotV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(
		util.ConcatSlice([][]byte{
			ib.BaseBallotV0.Bytes(),
			ib.INITBallotFactV0.Bytes(),
		}),
	), nil
}

func (ib INITBallotV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ib.INITBallotFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	var vb []byte
	if ib.Height() != Height(0) {
		vb = ib.voteproof.Bytes()
	}

	return valuehash.NewSHA256(
		util.ConcatSlice([][]byte{
			ib.INITBallotFactV0.Bytes(),
			vb,
		}),
	), nil
}

func (ib INITBallotV0) Fact() operation.Fact {
	return ib.INITBallotFactV0
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
	if h, err := ib.GenerateBodyHash(); err != nil {
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
	factHash := ib.INITBallotFactV0.Hash()
	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	ib.BaseBallotV0.signer = pk.Publickey()
	ib.BaseBallotV0.signature = sig
	ib.BaseBallotV0.signedAt = localtime.Now()
	ib.BaseBallotV0.bodyHash = bodyHash
	ib.BaseBallotV0.factHash = factHash
	ib.BaseBallotV0.factSignature = factSig

	if h, err := ib.GenerateHash(); err != nil {
		return err
	} else {
		ib.BaseBallotV0 = ib.BaseBallotV0.SetHash(h)
	}

	return nil
}
