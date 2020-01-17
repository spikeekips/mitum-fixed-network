package hint

import (
	jsoniter "github.com/json-iterator/go"
)

var jsoni = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

type hintJSON struct {
	Type    Type    `json:"type"`
	Version Version `json:"version"`
}

func (ht Hint) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(hintJSON{
		Type:    ht.t,
		Version: ht.version,
	})
}

func (ht *Hint) UnmarshalJSON(b []byte) error {
	var h hintJSON
	if err := jsoniter.Unmarshal(b, &h); err != nil {
		return err
	}

	ht.t = h.Type
	ht.version = h.Version

	return nil
}
