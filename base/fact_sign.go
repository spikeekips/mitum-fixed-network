package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

type FactSign interface {
	util.Byter
	isvalid.IsValider
	hint.Hinter
	Signer() key.Publickey
	Signature() key.Signature
	SignedAt() time.Time
}

var (
	BaseFactSignType = hint.Type("base-fact-sign")
	BaseFactSignHint = hint.NewHint(BaseFactSignType, "v0.0.1")
)

type FactSignUpdater interface {
	AddFactSigns(...FactSign) (FactSignUpdater, error)
}

func NewBytesForFactSignature(fact Fact, b []byte) []byte {
	return util.ConcatBytesSlice(fact.Hash().Bytes(), b)
}

func NewFactSignature(signer key.Privatekey, fact Fact, b []byte) (key.Signature, error) {
	return signer.Sign(NewBytesForFactSignature(fact, b))
}

func IsValidFactSign(fact Fact, fs FactSign, b []byte) error {
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
	return BaseFactSign{signer: signer, signature: signature, signedAt: localtime.UTCNow()}
}

func RawBaseFactSign(signer key.Publickey, signature key.Signature, signedAt time.Time) BaseFactSign {
	return BaseFactSign{signer: signer, signature: signature, signedAt: signedAt}
}

func (BaseFactSign) Hint() hint.Hint {
	return BaseFactSignHint
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

func (fs BaseFactSign) SetSignedAt(t time.Time) BaseFactSign {
	fs.signedAt = t

	return fs
}

func (fs BaseFactSign) Bytes() []byte {
	return util.ConcatBytesSlice(
		fs.signer.Bytes(),
		fs.signature.Bytes(),
		localtime.NewTime(fs.signedAt).Bytes(),
	)
}

func (fs BaseFactSign) IsValid([]byte) error {
	if fs.signedAt.IsZero() {
		return isvalid.InvalidError.Errorf("empty SignedAt")
	}

	return isvalid.Check(
		[]isvalid.IsValider{
			fs.signer,
			fs.signature,
		},
		nil, false)
}
