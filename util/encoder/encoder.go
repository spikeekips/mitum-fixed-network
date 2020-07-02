package encoder

import "github.com/spikeekips/mitum/util/hint"

const EncoderAnalyzedTypeDefault = "default"

type Encoder interface {
	hint.Hinter
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Decode([]byte, interface{}) error
	Analyze(interface{}) error
	SetHintset(*hint.Hintset)
	DecodeByHint([]byte) (hint.Hinter, error)
	DecodeWithHint(hint.Hint, []byte) (hint.Hinter, error)
}
