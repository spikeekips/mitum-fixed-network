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
	hint.BaseHinter
	h                    valuehash.Hash
	bodyHash             valuehash.Hash
	signer               key.Publickey
	signature            key.Signature
	signedAt             time.Time
	GenerateBodyHashFunc func() (valuehash.Hash, error)
}

func NewBaseSealWithHint(ht hint.Hint) BaseSeal {
	return BaseSeal{BaseHinter: hint.NewBaseHinter(ht)}
}

func (sl BaseSeal) IsValid(networkID []byte) error {
	if err := sl.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	return IsValidSeal(sl, networkID)
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
