package tree

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseFixedTreeNodeJSONPacker struct {
	jsonenc.HintedHead
	IN uint64 `json:"index"`
	KY string `json:"key"`
	HS string `json:"hash"`
}

func (no BaseFixedTreeNode) MarshalJSON() ([]byte, error) {
	if len(no.key) < 1 {
		return jsonenc.Marshal(nil)
	}

	return jsonenc.Marshal(BaseFixedTreeNodeJSONPacker{
		HintedHead: jsonenc.NewHintedHead(no.Hint()),
		IN:         no.index,
		KY:         base58.Encode(no.key),
		HS:         base58.Encode(no.hash),
	})
}

type BaseFixedTreeNodeJSONUnpacker struct {
	IN uint64 `json:"index"`
	KY string `json:"key"`
	HS string `json:"hash"`
}

func (no *BaseFixedTreeNode) UnmarshalJSON(b []byte) error {
	var uno BaseFixedTreeNodeJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uno); err != nil {
		return err
	}

	no.index = uno.IN
	no.key = base58.Decode(uno.KY)
	no.hash = base58.Decode(uno.HS)

	return nil
}

type FixedTreeJSONPacker struct {
	jsonenc.HintedHead
	NS []FixedTreeNode `json:"nodes"`
}

func (tr FixedTree) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(FixedTreeJSONPacker{
		HintedHead: jsonenc.NewHintedHead(tr.Hint()),
		NS:         tr.nodes,
	})
}

type FixedTreeJSONUnpacker struct {
	NS json.RawMessage `json:"nodes"`
}

func (tr *FixedTree) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var utr FixedTreeJSONUnpacker
	if err := enc.Unmarshal(b, &utr); err != nil {
		return err
	}

	return tr.unpack(enc, utr.NS)
}
