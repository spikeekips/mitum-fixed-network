package valuehash

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/hint"
)

type JSONHash struct {
	Hint hint.Hint `json:"_hint"`
	Hash string    `json:"hash"`
}

func (jh *JSONHash) Bytes() []byte {
	return base58.Decode(jh.Hash)
}

func MarshalJSON(h Hash) ([]byte, error) {
	return json.Marshal(JSONHash{
		Hint: h.Hint(),
		Hash: base58.Encode(h.Bytes()),
	})
}

func UnmarshalJSON(b []byte) (JSONHash, error) {
	var jh JSONHash
	if err := json.Unmarshal(b, &jh); err != nil {
		return JSONHash{}, err
	}

	return jh, nil
}
