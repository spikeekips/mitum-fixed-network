package encoder

import "github.com/spikeekips/mitum/util/hint"

type Encoder interface {
	hint.Hinter
	Add(hint.Hinter) error
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Decode([]byte) (hint.Hinter, error)                    // NOTE decode hinted instance
	DecodeWithHint([]byte, hint.Hint) (hint.Hinter, error) // NOTE decode with hint
	DecodeSlice([]byte) ([]hint.Hinter, error)             // NOTE decode slice of hinted instance
	DecodeMap([]byte) (map[string]hint.Hinter, error)      // NOTE decode string key map of hinted instance
}
