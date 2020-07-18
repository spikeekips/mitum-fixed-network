// +build test

package isaac

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	KVOperationHint     = hint.MustHintWithType(hint.Type{0xff, 0xfb}, "0.0.1", "kv-operation-isaac")
	LongKVOperationHint = hint.MustHintWithType(hint.Type{0xff, 0xfc}, "0.0.1", "long-kv-operation-isaac")
)

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

func (kvo KVOperation) Process(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(valuehash.Hash, ...state.StateUpdater) error,
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
		return setState(kvo.Hash(), s)
	}
}

type LongKVOperation struct {
	KVOperation
	preProcess func(
		getState func(key string) (state.StateUpdater, bool, error),
		setState func(valuehash.Hash, ...state.StateUpdater) error,
	) error
}

func NewLongKVOperation(op KVOperation) LongKVOperation {
	op.BaseOperation = op.SetHint(LongKVOperationHint)

	return LongKVOperation{
		KVOperation: op,
	}
}

func (kvo LongKVOperation) Hint() hint.Hint {
	return LongKVOperationHint
}

func (kvo LongKVOperation) Process(
	getState func(string) (state.StateUpdater, bool, error),
	setState func(valuehash.Hash, ...state.StateUpdater) error,
) error {
	if kvo.preProcess != nil {
		if err := kvo.preProcess(getState, setState); err != nil {
			return err
		}
	}

	return kvo.KVOperation.Process(getState, setState)
}

var longKVOperationFuncMap = &sync.Map{}

func (kvo LongKVOperation) SetPreProcess(
	f func(
		getState func(key string) (state.StateUpdater, bool, error),
		setState func(valuehash.Hash, ...state.StateUpdater) error,
	) error,
) LongKVOperation {
	kvo.preProcess = f

	longKVOperationFuncMap.Store(kvo.Hash().String(), f)

	return kvo
}

func (kvo *LongKVOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var bo operation.BaseOperation
	if err := bo.UnpackJSON(b, enc); err != nil {
		return err
	}

	kvo.BaseOperation = bo

	f, found := longKVOperationFuncMap.Load(bo.Hash().String())
	if found {
		kvo.preProcess = f.(func(func(string) (state.StateUpdater, bool, error), func(valuehash.Hash, ...state.StateUpdater) error) error)
	}

	return nil
}

func (kvo *LongKVOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var bo operation.BaseOperation
	if err := bo.UnpackBSON(b, enc); err != nil {
		return err
	}

	kvo.BaseOperation = bo

	f, found := longKVOperationFuncMap.Load(bo.Hash().String())
	if found {
		kvo.preProcess = f.(func(func(string) (state.StateUpdater, bool, error), func(valuehash.Hash, ...state.StateUpdater) error) error)
	}

	return nil
}
