package util

import (
	"encoding/json"

	jsoniter "github.com/json-iterator/go"
)

var jsoni = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

func JSONMarshal(i interface{}) ([]byte, error) {
	return jsoni.Marshal(i)
}

func JSONMarshalIndent(i interface{}) ([]byte, error) {
	return json.MarshalIndent(i, "", "  ")
}

func ToString(i interface{}) string {
	b, _ := JSONMarshalIndent(i)
	return string(b)
}

func JSONUnmarshal(b []byte, i interface{}) error {
	return jsoni.Unmarshal(b, i)
}
