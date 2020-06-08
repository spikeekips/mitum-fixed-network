package valuehash

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type JSONHash struct {
	jsonenc.HintedHead
	Hash string `json:"hash"`
}

func marshalJSON(h Hash) ([]byte, error) {
	return jsonenc.Marshal(JSONHash{
		HintedHead: jsonenc.NewHintedHead(h.Hint()),
		Hash:       h.String(),
	})
}

func unmarshalJSON(b []byte) (JSONHash, error) {
	var jh JSONHash
	if err := jsonenc.Unmarshal(b, &jh); err != nil {
		return JSONHash{}, err
	}

	return jh, nil
}
