package state

import (
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

func (st StateV0) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(st.Hint()),
		bson.M{
			"hash":            st.h,
			"key":             st.key,
			"value":           st.value,
			"previous_block":  st.previousBlock,
			"operation_infos": st.operations,
		},
	))
}

type StateV0UnpackerBSON struct {
	H   bson.Raw   `bson:"hash"`
	K   string     `bson:"key"`
	V   bson.Raw   `bson:"value"`
	PB  bson.Raw   `bson:"previous_block"`
	OPS []bson.Raw `bson:"operation_infos"`
}

func (st *StateV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var ust StateV0UnpackerBSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	ops := make([][]byte, len(ust.OPS))
	for i, b := range ust.OPS {
		ops[i] = b
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, ust.PB, ops)
}

func (oi OperationInfoV0) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(oi.Hint()),
		bson.M{
			"operation": oi.oh,
			"seal":      oi.sh,
		},
	))
}

type OperationInfoV0UnpackerBSON struct {
	OH bson.Raw `bson:"operation"`
	SH bson.Raw `bson:"seal"`
}

func (oi *OperationInfoV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var uoi OperationInfoV0UnpackerBSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
