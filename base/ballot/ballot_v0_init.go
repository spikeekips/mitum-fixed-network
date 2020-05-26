package ballot

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	INITBallotV0Hint     hint.Hint = hint.MustHint(INITBallotType, "0.0.1")
	INITBallotFactV0Hint hint.Hint = hint.MustHint(INITBallotFactType, "0.0.1")
)

type INITBallotFactV0 struct {
	BaseBallotFactV0
	previousBlock valuehash.Hash
}

func NewINITBallotFactV0(
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
) INITBallotFactV0 {
	return INITBallotFactV0{
		BaseBallotFactV0: NewBaseBallotFactV0(height, round),
		previousBlock:    previousBlock,
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
	return util.ConcatBytesSlice(
		ibf.BaseBallotFactV0.Bytes(),
		ibf.previousBlock.Bytes(),
	)
}

func (ibf INITBallotFactV0) PreviousBlock() valuehash.Hash {
	return ibf.previousBlock
}

type INITBallotV0 struct {
	BaseBallotV0
	INITBallotFactV0
	voteproof base.Voteproof
}

func NewINITBallotV0(
	node base.Address,
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
	voteproof base.Voteproof,
) INITBallotV0 {
	return INITBallotV0{
		BaseBallotV0: NewBaseBallotV0(node),
		INITBallotFactV0: NewINITBallotFactV0(
			height,
			round,
			previousBlock,
		),
		voteproof: voteproof,
	}
}

func (ib INITBallotV0) Hash() valuehash.Hash {
	return ib.BaseBallotV0.Hash()
}

func (ib INITBallotV0) Hint() hint.Hint {
	return INITBallotV0Hint
}

func (ib INITBallotV0) Stage() base.Stage {
	return base.StageINIT
}

func (ib INITBallotV0) IsValid(b []byte) error {
	if ib.Height() == base.Height(0) {
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

func (ib INITBallotV0) Voteproof() base.Voteproof {
	return ib.voteproof
}

func (ib INITBallotV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(util.ConcatBytesSlice(ib.BaseBallotV0.Bytes(), ib.INITBallotFactV0.Bytes())), nil
}

func (ib INITBallotV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ib.INITBallotFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	var vb []byte
	if ib.Height() != base.Height(0) && ib.voteproof != nil {
		vb = ib.voteproof.Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(ib.INITBallotFactV0.Bytes(), vb)), nil
}

func (ib INITBallotV0) Fact() base.Fact {
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
	if s, err := pk.Sign(util.ConcatBytesSlice(bodyHash.Bytes(), b)); err != nil {
		return err
	} else {
		sig = s
	}

	// fact signature
	factHash := ib.INITBallotFactV0.Hash()
	factSig, err := pk.Sign(util.ConcatBytesSlice(factHash.Bytes(), b))
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
