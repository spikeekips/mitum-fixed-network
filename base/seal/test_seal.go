package seal

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"go.mongodb.org/mongo-driver/bson"
)

var dummySealHint = hint.MustHintWithType(hint.Type{0xff, 0x35}, "0.1", "dummy-seal")

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
		S:         util.UUID().String(),
		CreatedAt: localtime.Now(),
	}
}

func (ds DummySeal) IsValid([]byte) error {
	return nil
}

func (ds DummySeal) Hint() hint.Hint {
	return dummySealHint
}

func (ds DummySeal) Hash() valuehash.Hash {
	return ds.H
}

func (ds DummySeal) GenerateHash() (valuehash.Hash, error) {
	return ds.H, nil
}

func (ds DummySeal) BodyHash() valuehash.Hash {
	return ds.BH
}

func (ds DummySeal) GenerateBodyHash() (valuehash.Hash, error) {
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

type DummySealJSONPacker struct {
	encoder.JSONPackHintedHead
	PK        key.BTCPrivatekey
	H         valuehash.SHA256
	BH        valuehash.SHA256
	S         string
	CreatedAt localtime.JSONTime
}

func (ds DummySeal) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(DummySealJSONPacker{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ds.Hint()),
		PK:                 ds.PK,
		H:                  ds.H,
		BH:                 ds.BH,
		S:                  ds.S,
		CreatedAt:          localtime.NewJSONTime(ds.CreatedAt),
	})
}

func (ds *DummySeal) UnmarshalJSON(b []byte) error {
	var uds DummySealJSONPacker
	if err := util.JSONUnmarshal(b, &uds); err != nil {
		return err
	}

	ds.PK = uds.PK
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt.Time

	return nil
}

func (ds DummySeal) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(ds.Hint()),
		bson.M{
			"PK":        ds.PK,
			"H":         ds.H,
			"BH":        ds.BH,
			"S":         ds.S,
			"CreatedAt": ds.CreatedAt,
		},
	))
}

type DummySealBSONPacker struct {
	PK        key.BTCPrivatekey
	H         valuehash.SHA256
	BH        valuehash.SHA256
	S         string
	CreatedAt time.Time
}

func (ds *DummySeal) UnmarshalBSON(b []byte) error {
	var uds DummySealBSONPacker
	if err := bson.Unmarshal(b, &uds); err != nil {
		return err
	}

	ds.PK = uds.PK
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt

	return nil
}
