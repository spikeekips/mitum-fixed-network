package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type StateV0PackerJSON struct {
	jsonenc.HintedHead
	H   valuehash.Hash   `json:"hash"`
	K   string           `json:"key"`
	V   Value            `json:"value"`
	PB  valuehash.Hash   `json:"previous_block"`
	HT  base.Height      `json:"height"`
	CB  valuehash.Hash   `json:"current_block"`
	OPS []valuehash.Hash `json:"operations"`
}

func (st StateV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(StateV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(st.Hint()),
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
	H   valuehash.Bytes   `json:"hash"`
	K   string            `json:"key"`
	V   json.RawMessage   `json:"value"`
	PB  valuehash.Bytes   `json:"previous_block"`
	HT  base.Height       `json:"height"`
	CB  valuehash.Bytes   `json:"current_block"`
	OPS []valuehash.Bytes `json:"operations"`
}

func (st *StateV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ust StateV0UnpackerJSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, ust.PB, ust.HT, ust.CB, ust.OPS)
}
