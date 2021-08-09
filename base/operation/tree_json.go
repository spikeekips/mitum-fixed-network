package operation

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
)

type FixedTreeNodeJSONPacker struct {
	jsonenc.HintedHead
	IN uint64      `json:"index"`
	KY string      `json:"key"`
	HS string      `json:"hash"`
	IS bool        `json:"in_state"`
	RS ReasonError `json:"reason"`
}

func (no FixedTreeNode) MarshalJSON() ([]byte, error) {
	if len(no.Key()) < 1 {
		return jsonenc.Marshal(nil)
	}

	return jsonenc.Marshal(FixedTreeNodeJSONPacker{
		HintedHead: jsonenc.NewHintedHead(no.Hint()),
		IN:         no.Index(),
		KY:         base58.Encode(no.Key()),
		HS:         base58.Encode(no.Hash()),
		IS:         no.inState,
		RS:         no.reason,
	})
}

type FixedTreeNodeJSONUnpacker struct {
	IS bool            `json:"in_state"`
	RS json.RawMessage `json:"reason"`
}

func (no *FixedTreeNode) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubno tree.BaseFixedTreeNode
	if err := jsonenc.Unmarshal(b, &ubno); err != nil {
		return err
	}

	var uno FixedTreeNodeJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uno); err != nil {
		return err
	}

	return no.unpack(enc, ubno, uno.IS, uno.RS)
}

type BaseReasonErrorJSONPacker struct {
	jsonenc.HintedHead
	MS string                 `json:"msg"`
	DT map[string]interface{} `json:"data"`
}

func (e BaseReasonError) MarshalJSON() ([]byte, error) {
	if e.NError == nil {
		return nil, nil
	}

	return jsonenc.Marshal(BaseReasonErrorJSONPacker{
		HintedHead: jsonenc.NewHintedHead(e.Hint()),
		MS:         e.Msg(),
		DT:         e.data,
	})
}

type BaseReasonErrorJSONUnpacker struct {
	MS string                 `json:"msg"`
	DT map[string]interface{} `json:"data"`
}

func (e *BaseReasonError) UnmarshalJSON(b []byte) error {
	var ue BaseReasonErrorJSONUnpacker
	if err := jsonenc.Unmarshal(b, &ue); err != nil {
		return err
	}

	e.NError = util.NewError(ue.MS)
	e.msg = ue.MS
	e.data = ue.DT

	return nil
}
