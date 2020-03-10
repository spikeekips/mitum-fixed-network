package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type StateV0PackerJSON struct {
	encoder.JSONPackHintedHead
	K   string          `json:"key"`
	V   interface{}     `json:"value"`
	VH  valuehash.Hash  `json:"value_hash"`
	PB  valuehash.Hash  `json:"previous_block"`
	OPS []OperationInfo `json:"operation_infos"`
}

func (st StateV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(StateV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(st.Hint()),
		K:                  st.key,
		V:                  st.value,
		VH:                 st.valueHash,
		PB:                 st.previousBlock,
		OPS:                st.operations,
	})
}

type StateV0UnpackerJSON struct {
	K   string            `json:"key"`
	V   json.RawMessage   `json:"value"`
	VH  json.RawMessage   `json:"value_hash"`
	PB  json.RawMessage   `json:"previous_block"`
	OPS []json.RawMessage `json:"operation_infos"`
}

func (st *StateV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var ust StateV0UnpackerJSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	var valueHash, previousBlock valuehash.Hash
	if h, err := valuehash.Decode(enc, ust.VH); err != nil {
		return err
	} else {
		valueHash = h
	}

	if h, err := valuehash.Decode(enc, ust.PB); err != nil {
		return err
	} else {
		previousBlock = h
	}

	ops := make([]OperationInfo, len(ust.OPS))
	for i := range ust.OPS {
		if oi, err := DecodeOperationInfo(enc, ust.OPS[i]); err != nil {
			return err
		} else {
			ops[i] = oi
		}
	}

	st.key = ust.K
	st.value = ust.V
	st.valueHash = valueHash
	st.previousBlock = previousBlock
	st.operations = ops

	return nil
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

	var oh, sh valuehash.Hash
	if h, err := valuehash.Decode(enc, uoi.OH); err != nil {
		return err
	} else {
		oh = h
	}

	if h, err := valuehash.Decode(enc, uoi.SH); err != nil {
		return err
	} else {
		sh = h
	}

	oi.oh = oh
	oi.sh = sh

	return nil
}
