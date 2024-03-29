package seal

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

var dummySealHint = hint.NewHint(hint.Type("dummy-seal"), "v0.1")

type DummySeal struct {
	PK        key.Publickey
	H         valuehash.Hash
	BH        valuehash.Hash
	S         string
	CreatedAt time.Time
}

func NewDummySeal(pk key.Publickey) DummySeal {
	return DummySeal{
		PK:        pk,
		H:         valuehash.RandomSHA256(),
		BH:        valuehash.RandomSHA256(),
		S:         util.UUID().String(),
		CreatedAt: localtime.UTCNow(),
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

func (ds DummySeal) GenerateHash() valuehash.Hash {
	return ds.H
}

func (ds DummySeal) BodyHash() valuehash.Hash {
	return ds.BH
}

func (ds DummySeal) GenerateBodyHash() (valuehash.Hash, error) {
	return ds.BH, nil
}

func (ds DummySeal) Signer() key.Publickey {
	return ds.PK
}

func (ds DummySeal) Signature() key.Signature {
	return key.Signature([]byte("showme"))
}

func (ds DummySeal) SignedAt() time.Time {
	return ds.CreatedAt
}

type DummySealJSONPacker struct {
	jsonenc.HintedHead
	PK        key.Publickey
	H         valuehash.Hash
	BH        valuehash.Hash
	S         string
	CreatedAt localtime.Time
}

type DummySealJSONUnpacker struct {
	jsonenc.HintedHead
	PK        key.PublickeyDecoder
	H         valuehash.Bytes
	BH        valuehash.Bytes
	S         string
	CreatedAt localtime.Time
}

func (ds DummySeal) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(DummySealJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ds.Hint()),
		PK:         ds.PK,
		H:          ds.H,
		BH:         ds.BH,
		S:          ds.S,
		CreatedAt:  localtime.NewTime(ds.CreatedAt),
	})
}

func (ds *DummySeal) UnmarshalJSON(b []byte) error {
	var uds DummySealJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uds); err != nil {
		return err
	}

	signer, err := key.LoadBasePublickey(string(uds.PK.Body()))
	if err != nil {
		return err
	}

	ds.PK = signer
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
	PK        key.PublickeyDecoder
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

	signer, err := key.LoadBasePublickey(string(uds.PK.Body()))
	if err != nil {
		return err
	}

	ds.PK = signer
	ds.H = uds.H
	ds.BH = uds.BH
	ds.S = uds.S
	ds.CreatedAt = uds.CreatedAt

	return nil
}

func (sl *BaseSeal) SignWithTime(pk key.Privatekey, networkID []byte, t time.Time) error {
	sl.signer = pk.Publickey()
	sl.signedAt = t

	var err error
	sl.bodyHash, err = sl.GenerateBodyHash()
	if err != nil {
		return err
	}

	sl.signature, err = pk.Sign(util.ConcatBytesSlice(sl.bodyHash.Bytes(), networkID))
	if err != nil {
		return err
	}

	sl.h = sl.GenerateHash()

	return nil
}
