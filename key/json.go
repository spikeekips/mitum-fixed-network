package key

import (
	"fmt"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
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

func MarshalJSONKey(k hint.Hinter) ([]byte, error) {
	return util.JSONMarshal(&struct {
		encoder.JSONPackHintedHead
		K string `json:"key"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(k.Hint()),
		K:                  k.(fmt.Stringer).String(),
	})
}

func UnmarshalJSONKey(b []byte) (hint.Hint, string, error) {
	var k struct {
		encoder.JSONPackHintedHead
		K string `json:"key"`
	}
	if err := util.JSONUnmarshal(b, &k); err != nil {
		return hint.Hint{}, "", err
	}

	return k.H, k.K, nil
}
