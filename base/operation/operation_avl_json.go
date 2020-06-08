package operation

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type baseOperationAVLNodeJSON struct {
	K   []byte `json:"key"`
	HT  int16  `json:"height"`
	LF  []byte `json:"left_key"`
	LFH []byte `json:"left_hash"`
	RG  []byte `json:"right_key"`
	RGH []byte `json:"right_hash"`
	H   []byte `json:"hash"`
}

type OperationAVLNodePackerJSON struct {
	jsonenc.HintedHead
	baseOperationAVLNodeJSON
	OP Operation `json:"operation"`
}

func (em OperationAVLNode) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(OperationAVLNodePackerJSON{
		HintedHead: jsonenc.NewHintedHead(em.Hint()),
		baseOperationAVLNodeJSON: baseOperationAVLNodeJSON{
			K:   em.key,
			HT:  em.height,
			LF:  em.left,
			LFH: em.leftHash,
			RG:  em.right,
			RGH: em.rightHash,
			H:   em.h,
		},
		OP: em.op,
	})
}

type OperationAVLNodeUnpackerJSON struct {
	baseOperationAVLNodeJSON
	OP json.RawMessage `json:"operation"`
}

func (em *OperationAVLNode) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ue OperationAVLNodeUnpackerJSON
	if err := enc.Unmarshal(b, &ue); err != nil {
		return err
	}

	return em.unpack(enc, ue.K, ue.HT, ue.LF, ue.LFH, ue.RG, ue.RGH, ue.H, ue.OP)
}
