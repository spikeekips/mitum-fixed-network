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
	SIGNBallotV0Hint     hint.Hint = hint.MustHint(SIGNBallotType, "0.1")
	SIGNBallotFactV0Hint hint.Hint = hint.MustHint(SIGNBallotFactType, "0.1")
)

type SIGNBallotFactV0 struct {
	BaseBallotFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (sbf SIGNBallotFactV0) Hint() hint.Hint {
	return SIGNBallotFactV0Hint
}

func (sbf SIGNBallotFactV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sbf.BaseBallotFactV0,
		sbf.proposal,
		sbf.newBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (sbf SIGNBallotFactV0) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{sbf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (sbf SIGNBallotFactV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		sbf.BaseBallotFactV0.Bytes(),
		sbf.proposal.Bytes(),
		sbf.newBlock.Bytes(),
	})
}

func (sbf SIGNBallotFactV0) Proposal() valuehash.Hash {
	return sbf.proposal
}

func (sbf SIGNBallotFactV0) NewBlock() valuehash.Hash {
	return sbf.newBlock
}

type SIGNBallotV0 struct {
	BaseBallotV0
	SIGNBallotFactV0
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	factHash      valuehash.Hash
	factSignature key.Signature
}

func NewSIGNBallotV0FromLocalState(
	localState *LocalState,
	round Round,
	newBlock Block,
	b []byte,
) (SIGNBallotV0, error) {
	lastBlock := localState.LastBlock()
	if lastBlock == nil {
		return SIGNBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	sb := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		SIGNBallotFactV0: SIGNBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: lastBlock.Height() + 1,
				round:  round,
			},
			proposal: newBlock.Proposal(),
			newBlock: newBlock.Hash(),
		},
	}

	// TODO NetworkID must be given.
	if err := sb.Sign(localState.Node().Privatekey(), b); err != nil {
		return SIGNBallotV0{}, err
	}

	return sb, nil
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
		sb.SIGNBallotFactV0,
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
		sb.SIGNBallotFactV0.Bytes(),
		sb.bodyHash.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb SIGNBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := sb.SIGNBallotFactV0.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		sb.SIGNBallotFactV0.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (sb SIGNBallotV0) Fact() Fact {
	return sb.SIGNBallotFactV0
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
	if h, err := sb.SIGNBallotFactV0.Hash(b); err != nil {
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
