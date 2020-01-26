package key

import (
	"fmt"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
)

func PackKeyJSON(k hint.Hinter, _ *encoder.JSONEncoder) (interface{}, error) {
	return &struct {
		encoder.JSONPackHintedHead
		K string `json:"key"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(k.Hint()),
		K:                  k.(fmt.Stringer).String(),
	}, nil
}

func UnpackKeyJSON(b []byte, enc *encoder.JSONEncoder) (string, error) {
	var k struct {
		K string `json:"key"`
	}
	if err := enc.Unmarshal(b, &k); err != nil {
		return "", err
	}

	return k.K, nil
}
