package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *BaseFact) unpack(
	_ encoder.Encoder,
	ht hint.Hint,
	h valuehash.Bytes,
	height base.Height,
	round base.Round,
) error {
	fact.hint = ht
	fact.h = h
	fact.height = height
	fact.round = round

	return nil
}
