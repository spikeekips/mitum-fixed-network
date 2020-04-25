package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type StateV0PackerJSON struct {
	encoder.JSONPackHintedHead
	H   valuehash.Hash  `json:"hash"`
	K   string          `json:"key"`
	V   Value           `json:"value"`
	PB  valuehash.Hash  `json:"previous_block"`
	OPS []OperationInfo `json:"operation_infos"`
}

func (st StateV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(StateV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(st.Hint()),
		H:                  st.h,
		K:                  st.key,
		V:                  st.value,
		PB:                 st.previousBlock,
		OPS:                st.operations,
	})
}

type StateV0UnpackerJSON struct {
	H   json.RawMessage   `json:"hash"`
	K   string            `json:"key"`
	V   json.RawMessage   `json:"value"`
	PB  json.RawMessage   `json:"previous_block"`
	OPS []json.RawMessage `json:"operation_infos"`
}

func (st *StateV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var ust StateV0UnpackerJSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	ops := make([][]byte, len(ust.OPS))
	for i, b := range ust.OPS {
		ops[i] = b
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, ust.PB, ops)
}

type OperationInfoV0PackerJSON struct {
	encoder.JSONPackHintedHead
	OH valuehash.Hash `json:"operation"`
	SH valuehash.Hash `json:"seal"`
}

func (oi OperationInfoV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(OperationInfoV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(oi.Hint()),
		OH:                 oi.oh,
		SH:                 oi.sh,
	})
}

type OperationInfoV0UnpackerJSON struct {
	OH json.RawMessage `json:"operation"`
	SH json.RawMessage `json:"seal"`
}

func (oi *OperationInfoV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uoi OperationInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
