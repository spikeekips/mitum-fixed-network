//go:build test
// +build test

package operation

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	MaxKeyKVOperation   int = 100
	MaxValueKVOperation int = 100
)

var (
	KVOperationFactType = hint.Type("kv-operation-fact")
	KVOperationFactHint = hint.NewHint(KVOperationFactType, "v0.0.1")
	KVOperationType     = hint.Type("kv-operation")
	KVOperationHint     = hint.NewHint(KVOperationType, "v0.0.1")
)

type KVOperationFact struct {
	T []byte `json:"token" bson:"token"`
	K string `json:"key" bson:"key"`
	V []byte `json:"value" bson:"value"`
}

func (kvof KVOperationFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(kvof, b); err != nil {
		return err
	}

	if kvof.V != nil {
		if l := len(kvof.V); l > MaxValueKVOperation {
			return errors.Errorf("Value of KVOperation over limit; %d > %d", l, MaxValueKVOperation)
		}
	}

	return nil
}

func (kvof KVOperationFact) Hint() hint.Hint {
	return KVOperationFactHint
}

func (kvof KVOperationFact) Hash() valuehash.Hash {
	return valuehash.NewSHA256(kvof.Bytes())
}

func (kvof KVOperationFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		kvof.T,
		[]byte(kvof.K),
		kvof.V,
	)
}

func (kvof KVOperationFact) Token() []byte {
	return kvof.T
}

func (kvof KVOperationFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(
		struct {
			jsonenc.HintedHead
			H valuehash.Hash `json:"hash"`
			T []byte         `json:"token"`
			K string         `json:"key"`
			V []byte         `json:"value"`
		}{
			HintedHead: jsonenc.NewHintedHead(kvof.Hint()),
			H:          kvof.Hash(),
			T:          kvof.T,
			K:          kvof.K,
			V:          kvof.V,
		},
	)
}

func (kvof KVOperationFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(kvof.Hint()),
		bson.M{
			"hash":  kvof.Hash(),
			"token": kvof.T,
			"key":   kvof.K,
			"value": kvof.V,
		},
	))
}

type KVOperation struct {
	BaseOperation
}

func NewKVOperation(
	signer key.Privatekey,
	token []byte,
	k string,
	v []byte,
	b []byte,
) (KVOperation, error) {
	fact := KVOperationFact{T: token, K: k, V: v}

	var fs []base.FactSign
	if sig, err := signer.Sign(util.ConcatBytesSlice(fact.Hash().Bytes(), b)); err != nil {
		return KVOperation{}, err
	} else {
		fs = []base.FactSign{base.NewBaseFactSign(signer.Publickey(), sig)}
	}

	if bo, err := NewBaseOperationFromFact(KVOperationHint, fact, fs); err != nil {
		return KVOperation{}, err
	} else {
		return KVOperation{BaseOperation: bo}, nil
	}
}

func (kvo KVOperation) Hint() hint.Hint {
	return KVOperationHint
}

func (kvo KVOperation) IsValid(networkID []byte) error {
	return IsValidOperation(kvo, networkID)
}

func (kvo KVOperation) Key() string {
	return kvo.Fact().(KVOperationFact).K
}

func (kvo KVOperation) Value() []byte {
	return kvo.Fact().(KVOperationFact).V
}

func (kvo KVOperation) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(kvo.BaseOperation)
}

func (kvo *KVOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var bo BaseOperation
	if err := bo.UnpackJSON(b, enc); err != nil {
		return err
	}

	kvo.BaseOperation = bo

	return nil
}

func (kvo KVOperation) MarshalBSON() ([]byte, error) {
	return bson.Marshal(kvo.BaseOperation)
}

func (kvo *KVOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var bo BaseOperation
	if err := bo.UnpackBSON(b, enc); err != nil {
		return err
	}

	kvo.BaseOperation = bo

	return nil
}
