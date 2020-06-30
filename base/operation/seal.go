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

// TODO MaxOperationsInSeal will be managed by Policy.

var MaxOperationsInSeal int = 100

var (
	SealType = hint.MustNewType(0x09, 0x00, "seal")
	SealHint = hint.MustHint(SealType, "0.0.1")
)

type Seal struct {
	h         valuehash.Hash
	bodyHash  valuehash.Hash
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
	ops       []Operation
}

func NewSeal(pk key.Privatekey, ops []Operation, networkID []byte) (Seal, error) {
	if len(ops) < 1 {
		return Seal{}, xerrors.Errorf("seal can not be generated without Operations")
	}

	sl := Seal{ops: ops}
	if err := sl.Sign(pk, networkID); err != nil {
		return Seal{}, err
	}

	return sl, nil
}

func (sl Seal) IsValid(networkID []byte) error {
	if l := len(sl.ops); l < 1 {
		return isvalid.InvalidError.Errorf("empty operations")
	} else if l > MaxOperationsInSeal {
		return isvalid.InvalidError.Errorf("operations over limit; %d > %d", l, MaxOperationsInSeal)
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

func (sl Seal) Hint() hint.Hint {
	return SealHint
}

func (sl Seal) Hash() valuehash.Hash {
	return sl.h
}

func (sl Seal) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(util.ConcatBytesSlice(sl.bodyHash.Bytes(), sl.signature.Bytes())), nil
}

func (sl Seal) BodyHash() valuehash.Hash {
	return sl.bodyHash
}

func (sl Seal) GenerateBodyHash() (valuehash.Hash, error) {
	bl := [][]byte{
		[]byte(sl.signer.String()),
		[]byte(localtime.RFC3339(sl.signedAt)),
	}

	for _, op := range sl.ops {
		bl = append(bl, op.Hash().Bytes())
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bl...)), nil
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

func (sl Seal) OperationHashes() []valuehash.Hash {
	hs := make([]valuehash.Hash, len(sl.ops))
	for i := range sl.ops {
		hs[i] = sl.ops[i].Hash()
	}

	return hs
}

func (sl *Seal) Sign(pk key.Privatekey, b []byte) error {
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

	if h, err := sl.GenerateHash(); err != nil {
		return err
	} else {
		sl.h = h
	}

	return nil
}
