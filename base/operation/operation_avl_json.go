package operation

import (
	"encoding/json"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
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
	encoder.JSONPackHintedHead
	baseOperationAVLNodeJSON
	OP Operation `json:"operation"`
}

func (em OperationAVLNode) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(OperationAVLNodePackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(em.Hint()),
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

func (em *OperationAVLNode) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var ue OperationAVLNodeUnpackerJSON
	if err := enc.Unmarshal(b, &ue); err != nil {
		return err
	}

	var op Operation
	if o, err := DecodeOperation(enc, ue.OP); err != nil {
		return err
	} else {
		op = o
	}

	em.key = ue.K
	em.height = ue.HT
	em.left = ue.LF
	em.leftHash = ue.LFH
	em.right = ue.RG
	em.rightHash = ue.RGH
	em.h = ue.H
	em.op = op

	return nil
}
