package seal

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
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
	jsonencoder.HintedHead
	PK        key.BTCPrivatekey
	H         valuehash.SHA256
	BH        valuehash.SHA256
	S         string
	CreatedAt localtime.JSONTime
}

func (ds DummySeal) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(DummySealJSONPacker{
		HintedHead: jsonencoder.NewHintedHead(ds.Hint()),
		PK:         ds.PK,
		H:          ds.H,
		BH:         ds.BH,
		S:          ds.S,
		CreatedAt:  localtime.NewJSONTime(ds.CreatedAt),
	})
}

func (ds *DummySeal) UnmarshalJSON(b []byte) error {
	var uds DummySealJSONPacker
	if err := jsonencoder.Unmarshal(b, &uds); err != nil {
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
	return bsonencoder.Marshal(bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(ds.Hint()),
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
	if err := bsonencoder.Unmarshal(b, &uds); err != nil {
		return err
	}

	ds.PK = uds.PK
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt

	return nil
}
