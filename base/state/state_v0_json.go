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

	var h, previousBlock valuehash.Hash
	if i, err := valuehash.Decode(enc, ust.H); err != nil {
		return err
	} else {
		h = i
	}

	if i, err := valuehash.Decode(enc, ust.PB); err != nil {
		return err
	} else {
		previousBlock = i
	}

	var value Value
	if v, err := DecodeValue(enc, ust.V); err != nil {
		return err
	} else {
		value = v
	}

	ops := make([]OperationInfo, len(ust.OPS))
	for i := range ust.OPS {
		if oi, err := DecodeOperationInfo(enc, ust.OPS[i]); err != nil {
			return err
		} else {
			ops[i] = oi
		}
	}

	st.h = h
	st.key = ust.K
	st.value = value
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
