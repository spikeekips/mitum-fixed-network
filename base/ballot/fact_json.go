package ballot

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseFactPackerJSON struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	HT base.Height    `json:"height"`
	R  base.Round     `json:"round"`
}

func (fact BaseFact) packerJSON() *BaseFactPackerJSON {
	return &BaseFactPackerJSON{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		HT:         fact.height,
		R:          fact.round,
	}
}

type BaseFactUnpackerJSON struct {
	HI hint.Hint       `json:"_hint"`
	H  valuehash.Bytes `json:"hash"`
	HT base.Height     `json:"height"`
	R  base.Round      `json:"round"`
}
