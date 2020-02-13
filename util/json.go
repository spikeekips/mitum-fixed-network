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

func MustJSONMarshal(i interface{}) []byte {
	b, _ := JSONMarshal(i)

	return b
}

func JSONMarshalIndent(i interface{}) ([]byte, error) {
	return json.MarshalIndent(i, "", "  ")
}

func MustJSONMarshalIndent(i interface{}) []byte {
	b, _ := JSONMarshalIndent(i)

	return b
}

func ToString(i interface{}) string {
	return string(MustJSONMarshalIndent(i))
}

func JSONUnmarshal(b []byte, i interface{}) error {
	return jsoni.Unmarshal(b, i)
}
