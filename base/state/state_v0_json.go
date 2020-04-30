package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type StateV0PackerJSON struct {
	jsonencoder.HintedHead
	H   valuehash.Hash  `json:"hash"`
	K   string          `json:"key"`
	V   Value           `json:"value"`
	PB  valuehash.Hash  `json:"previous_block"`
	HT  base.Height     `json:"height"`
	CB  valuehash.Hash  `json:"current_block"`
	OPS []OperationInfo `json:"operation_infos"`
}

func (st StateV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(StateV0PackerJSON{
		HintedHead: jsonencoder.NewHintedHead(st.Hint()),
		H:          st.h,
		K:          st.key,
		V:          st.value,
		PB:         st.previousBlock,
		HT:         st.currentHeight,
		CB:         st.currentBlock,
		OPS:        st.operations,
	})
}

type StateV0UnpackerJSON struct {
	H   json.RawMessage   `json:"hash"`
	K   string            `json:"key"`
	V   json.RawMessage   `json:"value"`
	PB  json.RawMessage   `json:"previous_block"`
	HT  base.Height       `json:"height"`
	CB  json.RawMessage   `json:"current_block"`
	OPS []json.RawMessage `json:"operation_infos"`
}

func (st *StateV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var ust StateV0UnpackerJSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	ops := make([][]byte, len(ust.OPS))
	for i, b := range ust.OPS {
		ops[i] = b
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, ust.PB, ust.HT, ust.CB, ops)
}

type OperationInfoV0PackerJSON struct {
	jsonencoder.HintedHead
	OH valuehash.Hash `json:"operation"`
	SH valuehash.Hash `json:"seal"`
}

func (oi OperationInfoV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(OperationInfoV0PackerJSON{
		HintedHead: jsonencoder.NewHintedHead(oi.Hint()),
		OH:         oi.oh,
		SH:         oi.sh,
	})
}

type OperationInfoV0UnpackerJSON struct {
	OH json.RawMessage `json:"operation"`
	SH json.RawMessage `json:"seal"`
}

func (oi *OperationInfoV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var uoi OperationInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
