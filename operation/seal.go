package operation

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

// TODO MaxOperationsInSeal will be managed by Policy.
var MaxOperationsInSeal int = 100

var SealHint hint.Hint = hint.MustHint(hint.Type{0x09, 0x00}, "0.0.1")

type Seal struct {
	h         valuehash.Hash
	bodyHash  valuehash.Hash
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
	ops       []Operation
}

func NewSeal(pk key.Privatekey, ops []Operation, b []byte) (Seal, error) {
	if len(ops) < 1 {
		return Seal{}, xerrors.Errorf("seal can not be generated without Operations")
	}

	sl := Seal{ops: ops}
	if err := sl.Sign(pk, b); err != nil {
		return Seal{}, err
	}

	return sl, nil
}

func (sl Seal) IsValid(b []byte) error {
	if l := len(sl.ops); l < 1 {
		return xerrors.Errorf("empty operations")
	} else if l > MaxOperationsInSeal {
		return xerrors.Errorf("operations over limit; %d > %d", l, MaxOperationsInSeal)
	}

	if err := seal.IsValidSeal(sl, b); err != nil {
		return err
	}

	for _, op := range sl.ops {
		if err := op.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}

func (sl Seal) Hint() hint.Hint {
	return SealHint
}

func (sl Seal) Hash() valuehash.Hash {
	return sl.h
}

func (sl Seal) GenerateHash(b []byte) (valuehash.Hash, error) {
	bl := [][]byte{
		sl.bodyHash.Bytes(),
		sl.signature.Bytes(),
	}

	bl = append(bl, b)

	return valuehash.NewSHA256(util.ConcatSlice(bl)), nil
}

func (sl Seal) BodyHash() valuehash.Hash {
	return sl.bodyHash
}

func (sl Seal) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	bl := [][]byte{
		[]byte(sl.signer.String()),
		[]byte(localtime.RFC3339(sl.signedAt)),
	}

	for _, op := range sl.ops {
		bl = append(bl, op.Hash().Bytes())
	}

	bl = append(bl, b)

	return valuehash.NewSHA256(util.ConcatSlice(bl)), nil
}

func (sl Seal) Signer() key.Publickey {
	return sl.signer
}

func (sl Seal) Signature() key.Signature {
	return sl.signature
}

func (sl Seal) SignedAt() time.Time {
	return sl.signedAt
}

func (sl Seal) Operations() []Operation {
	return sl.ops
}

func (sl *Seal) Sign(pk key.Privatekey, b []byte) error {
	sl.signer = pk.Publickey()
	sl.signedAt = localtime.Now()

	var bodyHash valuehash.Hash
	if h, err := sl.GenerateBodyHash(b); err != nil {
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

	sl.signature = sig
	sl.bodyHash = bodyHash

	if h, err := sl.GenerateHash(b); err != nil {
		return err
	} else {
		sl.h = h
	}

	return nil
}
