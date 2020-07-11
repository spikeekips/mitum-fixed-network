// +build test

package isaac

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var KVOperationHint = hint.MustHintWithType(hint.Type{0xff, 0xfb}, "0.0.1", "kv-operation-isaac")

type KVOperation struct {
	operation.KVOperation
}

func NewKVOperation(
	signer key.Privatekey,
	token []byte,
	k string,
	v []byte,
	b []byte,
) (KVOperation, error) {
	op, err := operation.NewKVOperation(signer, token, k, v, b)
	if err != nil {
		return KVOperation{}, err
	}

	op.BaseOperation = op.SetHint(KVOperationHint)

	return KVOperation{
		KVOperation: op,
	}, nil
}

func (kvo KVOperation) Hint() hint.Hint {
	return KVOperationHint
}

func (kvo KVOperation) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(kvo.BaseOperation)
}

func (kvo *KVOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var bo operation.BaseOperation
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
	var bo operation.BaseOperation
	if err := bo.UnpackBSON(b, enc); err != nil {
		return err
	}

	kvo.BaseOperation = bo

	return nil
}

func (kvo KVOperation) ProcessOperation(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(...state.StateUpdater) error,
) error {
	var value state.BytesValue
	if v, err := state.NewBytesValue(kvo.Value()); err != nil {
		return err
	} else {
		value = v
	}

	if s, _, err := getState(kvo.Key()); err != nil {
		return err
	} else if err := s.SetValue(value); err != nil {
		return err
	} else {
		return setState(s)
	}
}
