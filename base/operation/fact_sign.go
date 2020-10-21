package operation

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"golang.org/x/xerrors"
)

var (
	baseFactSignType = hint.MustNewType(0x01, 0x50, "base-fact-sign")
	baseFactSignHint = hint.MustHint(baseFactSignType, "0.0.1")
)

type FactSignUpdater interface {
	AddFactSigns(...FactSign) (FactSignUpdater, error)
}

type FactSign interface {
	util.Byter
	isvalid.IsValider
	hint.Hinter
	Signer() key.Publickey
	Signature() key.Signature
	SignedAt() time.Time
}

func NewBytesForFactSignature(fact base.Fact, b []byte) []byte {
	return util.ConcatBytesSlice(fact.Hash().Bytes(), b)
}

func NewFactSignature(signer key.Privatekey, fact base.Fact, b []byte) (key.Signature, error) {
	if fs, err := signer.Sign(NewBytesForFactSignature(fact, b)); err != nil {
		return nil, err
	} else {
		return fs, nil
	}
}

func IsValidFactSign(fact base.Fact, fs FactSign, b []byte) error {
	if fact == nil || fact.Hash() == nil {
		return isvalid.InvalidError.Errorf("empty Fact")
	}
	if fs == nil {
		return isvalid.InvalidError.Errorf("empty FactSign")
	}
	if fs.Signer() == nil {
		return isvalid.InvalidError.Errorf("FactSign has empty Signer()")
	}
	if fs.Signature() == nil {
		return isvalid.InvalidError.Errorf("FactSign has empty Signature()")
	}

	if err := fs.Signer().Verify(util.ConcatBytesSlice(fact.Hash().Bytes(), b), fs.Signature()); err != nil {
		return err
	}

	return fs.Signer().Verify(
		util.ConcatBytesSlice(fact.Hash().Bytes(), b),
		fs.Signature(),
	)
}

type BaseFactSign struct {
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
}

func NewBaseFactSign(signer key.Publickey, signature key.Signature) BaseFactSign {
	return BaseFactSign{signer: signer, signature: signature, signedAt: localtime.Now()}
}

func RawBaseFactSign(signer key.Publickey, signature key.Signature, signedAt time.Time) BaseFactSign {
	return BaseFactSign{signer: signer, signature: signature, signedAt: signedAt}
}

func (fs BaseFactSign) Hint() hint.Hint {
	return baseFactSignHint
}

func (fs BaseFactSign) Signer() key.Publickey {
	return fs.signer
}

func (fs BaseFactSign) Signature() key.Signature {
	return fs.signature
}

func (fs BaseFactSign) SignedAt() time.Time {
	return fs.signedAt
}

func (fs BaseFactSign) Bytes() []byte {
	return util.ConcatBytesSlice(
		fs.signer.Bytes(),
		fs.signature.Bytes(),
		localtime.NewTime(fs.signedAt).Bytes(),
	)
}

func (fs BaseFactSign) IsValid(b []byte) error {
	if fs.signedAt.IsZero() {
		return xerrors.Errorf("empty SignedAt")
	}

	return isvalid.Check(
		[]isvalid.IsValider{
			fs.signer,
			fs.signature,
		},
		b, false)
}
