package valuehash

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/encoder"
)

type JSONHash struct {
	encoder.JSONPackHintedHead
	Hash string `json:"hash"`
}

func (jh *JSONHash) Bytes() []byte {
	return base58.Decode(jh.Hash)
}

func MarshalJSON(h Hash) ([]byte, error) {
	return json.Marshal(JSONHash{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(h.Hint()),
		Hash:               base58.Encode(h.Bytes()),
	})
}

func UnmarshalJSON(b []byte) (JSONHash, error) {
	var jh JSONHash
	if err := json.Unmarshal(b, &jh); err != nil {
		return JSONHash{}, err
	}

	return jh, nil
}
