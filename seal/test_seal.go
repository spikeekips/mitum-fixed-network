package seal

import (
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type DummySeal struct {
	PK        key.BTCPrivatekey
	H         valuehash.SHA256
	BH        valuehash.SHA256
	S         string
	CreatedAt time.Time
}

func NewDummySeal(pk key.BTCPrivatekey) DummySeal {
	return DummySeal{
		PK:        pk,
		H:         valuehash.RandomSHA256().(valuehash.SHA256),
		BH:        valuehash.RandomSHA256().(valuehash.SHA256),
		S:         uuid.Must(uuid.NewV4(), nil).String(),
		CreatedAt: localtime.Now(),
	}
}

func (ds DummySeal) IsValid([]byte) error {
	return nil
}

func (ds DummySeal) Hint() hint.Hint {
	return hint.MustHint(hint.Type{0xff, 0x30}, "0.1")
}

func (ds DummySeal) Hash() valuehash.Hash {
	return ds.H
}

func (ds DummySeal) GenerateHash([]byte) (valuehash.Hash, error) {
	return ds.H, nil
}

func (ds DummySeal) BodyHash() valuehash.Hash {
	return ds.BH
}

func (ds DummySeal) GenerateBodyHash([]byte) (valuehash.Hash, error) {
	return ds.BH, nil
}

func (ds DummySeal) Signer() key.Publickey {
	return ds.PK.Publickey()
}

func (ds DummySeal) Signature() key.Signature {
	return key.Signature([]byte("showme"))
}

func (ds DummySeal) SignedAt() time.Time {
	return ds.CreatedAt
}

func (ds DummySeal) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		PK        key.Privatekey
		H         valuehash.Hash
		BH        valuehash.Hash
		CreatedAt time.Time
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ds.Hint()),
		PK:                 ds.PK,
		H:                  ds.H,
		BH:                 ds.BH,
		CreatedAt:          ds.CreatedAt,
	})
}
