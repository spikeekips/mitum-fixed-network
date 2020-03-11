package valuehash

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
)

type JSONHash struct {
	encoder.JSONPackHintedHead
	Hash string `json:"hash"`
}

func marshalJSON(h Hash) ([]byte, error) {
	return util.JSONMarshal(JSONHash{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(h.Hint()),
		Hash:               h.String(),
	})
}

func unmarshalJSON(b []byte) (JSONHash, error) {
	var jh JSONHash
	if err := util.JSONUnmarshal(b, &jh); err != nil {
		return JSONHash{}, err
	}

	return jh, nil
}
