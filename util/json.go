package util

import jsoniter "github.com/json-iterator/go"

var jsoni = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

func JSONMarshal(i interface{}) ([]byte, error) {
	return jsoni.Marshal(i)
}

func JSONUnmarshal(b []byte, i interface{}) error {
	return jsoni.Unmarshal(b, i)
}
