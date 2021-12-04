package block

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
)

type BlockV0PackJSON struct {
	jsonenc.HintedHead
	MF  ManifestV0            `json:"manifest"`
	CI  ConsensusInfoV0       `json:"consensus"`
	OPT tree.FixedTree        `json:"operations_tree"`
	OP  []operation.Operation `json:"operations"`
	STT tree.FixedTree        `json:"states_tree"`
	ST  []state.State         `json:"states"`
}

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BlockV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(bm.Hint()),
		MF:         bm.ManifestV0,
		CI:         bm.ci,
		OPT:        bm.operationsTree,
		OP:         bm.operations,
		STT:        bm.statesTree,
		ST:         bm.states,
	})
}

type BlockV0UnpackJSON struct {
	MF  json.RawMessage `json:"manifest"`
	CI  json.RawMessage `json:"consensus"`
	OPT json.RawMessage `json:"operations_tree"`
	OP  json.RawMessage `json:"operations"`
	STT json.RawMessage `json:"states_tree"`
	ST  json.RawMessage `json:"states"`
}

func (bm *BlockV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var um BlockV0UnpackJSON
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	}

	return bm.unpack(enc, um.MF, um.CI, um.OPT, um.OP, um.STT, um.ST)
}
