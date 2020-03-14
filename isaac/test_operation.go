package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util"
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

	return KVOperation{
		KVOperation: op,
	}, nil
}

func (kvo KVOperation) Hint() hint.Hint {
	return KVOperationHint
}

func (kvo KVOperation) MarshalJSON() ([]byte, error) {
	b, err := util.JSONMarshal(kvo.KVOperation)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := util.JSONUnmarshal(b, &m); err != nil {
		return nil, err
	} else {
		m["_hint"] = kvo.Hint()
	}

	return util.JSONMarshal(m)
}

func (kvo *KVOperation) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	okvo := &operation.KVOperation{}
	if err := okvo.UnpackJSON(b, enc); err != nil {
		return err
	}

	kvo.KVOperation = *okvo

	return nil
}

func (kvo KVOperation) ProcessOperation(
	getState func(key string) (state.StateUpdater, error),
	setState func(state.StateUpdater) error,
) (state.StateUpdater, error) {
	value, err := state.NewBytesValue(kvo.Value)
	if err != nil {
		return nil, err
	}

	var st state.StateUpdater
	if s, err := getState(kvo.Key); err != nil {
		return nil, err
	} else if err := s.SetValue(value); err != nil {
		return nil, err
	} else {
		st = s
	}

	return st, setState(st)
}
