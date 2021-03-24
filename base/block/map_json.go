package block

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseBlockDataMapJSONPacker struct {
	jsonenc.HintedHead
	H         valuehash.Hash                  `json:"hash"`
	Height    base.Height                     `json:"height"`
	Block     valuehash.Hash                  `json:"block"`
	CreatedAt time.Time                       `json:"created_at"`
	Items     map[string]BaseBlockDataMapItem `json:"items"`
	Writer    hint.Hint                       `json:"writer"`
}

func (bd BaseBlockDataMap) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseBlockDataMapJSONPacker{
		HintedHead: jsonenc.NewHintedHead(bd.Hint()),
		H:          bd.h,
		Height:     bd.height,
		Block:      bd.block,
		CreatedAt:  bd.createdAt,
		Items:      bd.items,
		Writer:     bd.writerHint,
	})
}

type BaseBlockDataMapJSONUnpacker struct {
	H         valuehash.Bytes            `json:"hash"`
	Height    base.Height                `json:"height"`
	Block     valuehash.Bytes            `json:"block"`
	CreatedAt localtime.Time             `json:"created_at"`
	Items     map[string]json.RawMessage `json:"items"`
	Writer    hint.Hint                  `json:"writer"`
}

func (bd *BaseBlockDataMap) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubd BaseBlockDataMapJSONUnpacker
	if err := enc.Unmarshal(b, &ubd); err != nil {
		return err
	}

	bitems := map[string][]byte{}
	for k := range ubd.Items {
		bitems[k] = ubd.Items[k]
	}

	return bd.unpack(enc, ubd.H, ubd.Height, ubd.Block, ubd.CreatedAt.Time, bitems, ubd.Writer)
}

type BaseBlockDataMapItemJSONPacker struct {
	Type     string `json:"type"`
	Checksum string `json:"checksum"`
	URL      string `json:"url"`
}

func (bd BaseBlockDataMapItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseBlockDataMapItemJSONPacker{
		Type:     bd.t,
		Checksum: bd.checksum,
		URL:      bd.url,
	})
}

func (bd *BaseBlockDataMapItem) UnmarshalJSON(b []byte) error {
	var ubdi BaseBlockDataMapItemJSONPacker
	if err := jsonenc.Unmarshal(b, &ubdi); err != nil {
		return err
	}

	return bd.unpack(ubdi.Type, ubdi.Checksum, ubdi.URL)
}
