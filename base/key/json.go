package key

import (
	"fmt"

	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

func MarshalJSONKey(k hint.Hinter) ([]byte, error) {
	return jsonencoder.Marshal(&struct {
		jsonencoder.HintedHead
		K string `json:"key"`
	}{
		HintedHead: jsonencoder.NewHintedHead(k.Hint()),
		K:          k.(fmt.Stringer).String(),
	})
}

func UnmarshalJSONKey(b []byte) (hint.Hint, string, error) {
	var k struct {
		jsonencoder.HintedHead
		K string `json:"key"`
	}
	if err := jsonencoder.Unmarshal(b, &k); err != nil {
		return hint.Hint{}, "", err
	}

	return k.H, k.K, nil
}
