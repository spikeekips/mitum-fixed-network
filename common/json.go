package common

import (
	"bytes"
	"encoding/json"
)

func EncodeJSON(v interface{}, indent, escapeHTML bool) ([]byte, error) {
	buffer := &bytes.Buffer{}
	e := json.NewEncoder(buffer)
	if indent {
		e.SetIndent("", "  ")
	}
	e.SetEscapeHTML(escapeHTML)

	err := e.Encode(v)
	if err != nil {
		return nil, err
	}

	return bytes.TrimRight(buffer.Bytes(), "\n"), err
}
