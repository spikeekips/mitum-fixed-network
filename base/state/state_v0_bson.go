package state

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (st StateV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(st.Hint()),
		bson.M{
			"hash":            st.h,
			"key":             st.key,
			"value":           st.value,
			"previous_block":  st.previousBlock,
			"height":          st.currentHeight,
			"current_block":   st.currentBlock,
			"operation_infos": st.operations,
		},
	))
}

type StateV0UnpackerBSON struct {
	H   valuehash.Bytes `bson:"hash"`
	K   string          `bson:"key"`
	V   bson.Raw        `bson:"value"`
	PB  bson.RawValue   `bson:"previous_block"`
	HT  base.Height     `bson:"height"`
	CB  valuehash.Bytes `bson:"current_block"`
	OPS []bson.Raw      `bson:"operation_infos"`
}

func (st *StateV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ust StateV0UnpackerBSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	var pb valuehash.Bytes
	if s, ok := ust.PB.StringValueOK(); ok && len(s) > 0 {
		if err := ust.PB.Unmarshal(&pb); err != nil {
			return err
		}
	}

	ops := make([][]byte, len(ust.OPS))
	for i, b := range ust.OPS {
		ops[i] = b
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, pb, ust.HT, ust.CB, ops)
}

func (oi OperationInfoV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(oi.Hint()),
		bson.M{
			"operation": oi.oh,
			"seal":      oi.sh,
		},
	))
}

type OperationInfoV0UnpackerBSON struct {
	OH valuehash.Bytes `bson:"operation"`
	SH valuehash.Bytes `bson:"seal"`
}

func (oi *OperationInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uoi OperationInfoV0UnpackerBSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
