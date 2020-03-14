package state

import (
	"encoding/json"
	"reflect"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type NumberValueJSONPacker struct {
	encoder.JSONPackHintedHead
	H valuehash.Hash `json:"hash"`
	V []byte         `json:"value"`
	T reflect.Kind   `json:"type"`
}

func (nv NumberValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(NumberValueJSONPacker{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(nv.Hint()),
		H:                  nv.Hash(),
		V:                  nv.b,
		T:                  nv.t,
	})
}

type NumberValueJSONUnpacker struct {
	H json.RawMessage `json:"hash"`
	V []byte          `json:"value"`
	T reflect.Kind    `json:"type"`
}

func (nv *NumberValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uv NumberValueJSONUnpacker
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	if i, err := valuehash.Decode(enc, uv.H); err != nil {
		return err
	} else {
		nv.h = i
	}

	var v interface{}
	switch uv.T {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := util.BytesToInt64(uv.V); err != nil {
			return err
		} else {
			switch uv.T {
			case reflect.Int:
				v = int(i)
			case reflect.Int8:
				v = int8(i)
			case reflect.Int16:
				v = int16(i)
			case reflect.Int32:
				v = int32(i)
			case reflect.Int64:
				v = i
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, err := util.BytesToUint64(uv.V); err != nil {
			return err
		} else {
			switch uv.T {
			case reflect.Uint:
				v = uint(i)
			case reflect.Uint8:
				v = uint8(i)
			case reflect.Uint16:
				v = uint16(i)
			case reflect.Uint32:
				v = uint32(i)
			case reflect.Uint64:
				v = i
			}
		}
	case reflect.Float64:
		v = util.BytesToFloat64(uv.V)
	default:
		return xerrors.Errorf("unsupported type for NumberValue: %v", uv.T)
	}

	nv.v = v
	nv.b = uv.V

	return nil
}
