package valuehash

import (
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type JSONHash struct {
	jsonencoder.HintedHead
	Hash string `json:"hash"`
}

func marshalJSON(h Hash) ([]byte, error) {
	return jsonencoder.Marshal(JSONHash{
		HintedHead: jsonencoder.NewHintedHead(h.Hint()),
		Hash:       h.String(),
	})
}

func unmarshalJSON(b []byte) (JSONHash, error) {
	var jh JSONHash
	if err := jsonencoder.Unmarshal(b, &jh); err != nil {
		return JSONHash{}, err
	}

	return jh, nil
}
