package seal

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseSeal struct {
	h                    valuehash.Hash
	ht                   hint.Hint
	bodyHash             valuehash.Hash
	signer               key.Publickey
	signature            key.Signature
	signedAt             time.Time
	GenerateBodyHashFunc func() (valuehash.Hash, error)
}

func NewBaseSealWithHint(ht hint.Hint) BaseSeal {
	return BaseSeal{ht: ht}
}

func NewBaseSeal(ht hint.Hint, pk key.Privatekey, networkID []byte) (BaseSeal, error) {
	sl := BaseSeal{ht: ht}
	if err := sl.Sign(pk, networkID); err != nil {
		return BaseSeal{}, err
	}

	return sl, nil
}

func (sl BaseSeal) IsValid(networkID []byte) error {
	return IsValidSeal(sl, networkID)
}

func (sl BaseSeal) Hint() hint.Hint {
	return sl.ht
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

func (sl BaseSeal) BodyBytes() []byte {
	return util.ConcatBytesSlice(
		sl.signer.Bytes(),
		localtime.NewTime(sl.signedAt).Bytes(),
	)
}

func (sl BaseSeal) GenerateBodyHash() (valuehash.Hash, error) {
	f := sl.defaultGenerateBodyHash
	if sl.GenerateBodyHashFunc != nil {
		f = sl.GenerateBodyHashFunc
	}

	return f()
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

func (sl *BaseSeal) Sign(pk key.Privatekey, b []byte) error {
	sl.signer = pk.Publickey()
	sl.signedAt = localtime.UTCNow()

	var err error
	sl.bodyHash, err = sl.GenerateBodyHash()
	if err != nil {
		return err
	}

	sl.signature, err = pk.Sign(util.ConcatBytesSlice(sl.bodyHash.Bytes(), b))
	if err != nil {
		return err
	}

	sl.h = sl.GenerateHash()

	return nil
}

func (sl BaseSeal) defaultGenerateBodyHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(sl.BodyBytes()), nil
}
