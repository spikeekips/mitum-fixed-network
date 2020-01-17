package encoder

import "github.com/spikeekips/mitum/hint"

const encoderAnalyzedTypeDefault = "default"

type Encoder interface {
	hint.Hinter
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
	Analyze(interface{}) error
	SetHintset(*hint.Hintset)
	DecodeByHint([]byte) (hint.Hinter, error)
}
