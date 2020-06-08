package key

import (
	"fmt"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

func MarshalJSONKey(k hint.Hinter) ([]byte, error) {
	return jsonenc.Marshal(&struct {
		jsonenc.HintedHead
		K string `json:"key"`
	}{
		HintedHead: jsonenc.NewHintedHead(k.Hint()),
		K:          k.(fmt.Stringer).String(),
	})
}

func UnmarshalJSONKey(b []byte) (hint.Hint, string, error) {
	var k struct {
		jsonenc.HintedHead
		K string `json:"key"`
	}
	if err := jsonenc.Unmarshal(b, &k); err != nil {
		return hint.Hint{}, "", err
	}

	return k.H, k.K, nil
}
