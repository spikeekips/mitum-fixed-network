package operation

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type OperationInfoV0PackerJSON struct {
	jsonenc.HintedHead
	OH valuehash.Hash `json:"operation"`
	SH valuehash.Hash `json:"seal"`
}

func (oi OperationInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(OperationInfoV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(oi.Hint()),
		OH:         oi.oh,
		SH:         oi.sh,
	})
}

type OperationInfoV0UnpackerJSON struct {
	OH valuehash.Bytes `json:"operation"`
	SH valuehash.Bytes `json:"seal"`
}

func (oi *OperationInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uoi OperationInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
