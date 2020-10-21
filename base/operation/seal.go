package operation

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Seal interface {
	seal.Seal
	Operations() []Operation
}

type SealUpdater interface {
	SetOperations([]Operation) SealUpdater
}

var (
	SealType = hint.MustNewType(0x01, 0x51, "seal")
	SealHint = hint.MustHint(SealType, "0.0.1")
)

type BaseSeal struct {
	h         valuehash.Hash
	bodyHash  valuehash.Hash
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
	ops       []Operation
}

func NewBaseSeal(pk key.Privatekey, ops []Operation, networkID []byte) (BaseSeal, error) {
	if len(ops) < 1 {
		return BaseSeal{}, xerrors.Errorf("seal can not be generated without Operations")
	}

	sl := BaseSeal{ops: ops}
	if err := sl.Sign(pk, networkID); err != nil {
		return BaseSeal{}, err
	}

	return sl, nil
}

func (sl BaseSeal) IsValid(networkID []byte) error {
	if l := len(sl.ops); l < 1 {
		return isvalid.InvalidError.Errorf("empty operations")
	}

	if err := seal.IsValidSeal(sl, networkID); err != nil {
		return err
	}

	for _, op := range sl.ops {
		if err := op.IsValid(networkID); err != nil {
			return err
		}
	}

	return nil
}

func (sl BaseSeal) Hint() hint.Hint {
	return SealHint
}

func (sl BaseSeal) Hash() valuehash.Hash {
	return sl.h
}

func (sl BaseSeal) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(util.ConcatBytesSlice(sl.bodyHash.Bytes(), sl.signature.Bytes()))
}

func (sl BaseSeal) BodyHash() valuehash.Hash {
	return sl.bodyHash
}

func (sl BaseSeal) GenerateBodyHash() (valuehash.Hash, error) {
	bl := [][]byte{
		sl.signer.Bytes(),
		localtime.NewTime(sl.signedAt).Bytes(),
	}

	for _, op := range sl.ops {
		bl = append(bl, op.Hash().Bytes())
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bl...)), nil
}

func (sl BaseSeal) Signer() key.Publickey {
	return sl.signer
}

func (sl BaseSeal) Signature() key.Signature {
	return sl.signature
}

func (sl BaseSeal) SignedAt() time.Time {
	return sl.signedAt
}

func (sl BaseSeal) Operations() []Operation {
	return sl.ops
}

func (sl BaseSeal) SetOperations(ops []Operation) SealUpdater {
	sl.ops = ops

	return sl
}

func (sl *BaseSeal) Sign(pk key.Privatekey, b []byte) error {
	sl.signer = pk.Publickey()
	sl.signedAt = localtime.Now()

	var bodyHash valuehash.Hash
	if h, err := sl.GenerateBodyHash(); err != nil {
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

	sl.signature = sig
	sl.bodyHash = bodyHash
	sl.h = sl.GenerateHash()

	return nil
}
