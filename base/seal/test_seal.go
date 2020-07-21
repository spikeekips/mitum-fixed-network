package seal

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var dummySealHint = hint.MustHintWithType(hint.Type{0xff, 0x35}, "0.1", "dummy-seal")

type DummySeal struct {
	PK        key.Privatekey
	H         valuehash.Hash
	BH        valuehash.Hash
	S         string
	CreatedAt time.Time
}

func NewDummySeal(pk key.Privatekey) DummySeal {
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
	jsonenc.HintedHead
	PK        key.Privatekey
	H         valuehash.Hash
	BH        valuehash.Hash
	S         string
	CreatedAt localtime.JSONTime
}

type DummySealJSONUnpacker struct {
	jsonenc.HintedHead
	PK        encoder.HintedString
	H         valuehash.Bytes
	BH        valuehash.Bytes
	S         string
	CreatedAt localtime.JSONTime
}

func (ds DummySeal) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(DummySealJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ds.Hint()),
		PK:         ds.PK,
		H:          ds.H,
		BH:         ds.BH,
		S:          ds.S,
		CreatedAt:  localtime.NewJSONTime(ds.CreatedAt),
	})
}

func (ds *DummySeal) UnmarshalJSON(b []byte) error {
	var uds DummySealJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uds); err != nil {
		return err
	}

	signer := new(key.BTCPrivatekey)
	if err := signer.UnmarshalText([]byte(uds.PK.String())); err != nil {
		return err
	}

	ds.PK = *signer
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt.Time

	return nil
}

func (ds DummySeal) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ds.Hint()),
		bson.M{
			"PK":        ds.PK,
			"H":         ds.H,
			"BH":        ds.BH,
			"S":         ds.S,
			"CreatedAt": ds.CreatedAt,
		},
	))
}

type DummySealBSONUnpacker struct {
	PK        encoder.HintedString
	H         valuehash.Bytes
	BH        valuehash.Bytes
	S         string
	CreatedAt time.Time
}

func (ds *DummySeal) UnmarshalBSON(b []byte) error {
	var uds DummySealBSONUnpacker
	if err := bsonenc.Unmarshal(b, &uds); err != nil {
		return err
	}

	signer := new(key.BTCPrivatekey)
	if err := signer.UnmarshalText([]byte(uds.PK.String())); err != nil {
		return err
	}

	ds.PK = *signer
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt

	return nil
}
