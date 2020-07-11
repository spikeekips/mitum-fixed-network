package util

import jsoniter "github.com/json-iterator/go"

var JSON = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: false,
}.Froze()
