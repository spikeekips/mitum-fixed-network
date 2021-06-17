package jsonenc

import (
	"github.com/spikeekips/mitum/util/hint"
)

type HintedHead struct {
	H hint.Hint `json:"_hint"`
}

func NewHintedHead(h hint.Hint) HintedHead {
	return HintedHead{H: h}
}
