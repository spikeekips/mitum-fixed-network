package ballot

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (sl *BaseSeal) unpack(
	enc encoder.Encoder,
	bf,
	bbb,
	bba []byte,
) error {
	if err := encoder.Decode(bf, enc, &sl.sfs); err != nil {
		return err
	}

	if err := encoder.Decode(bbb, enc, &sl.baseVoteproof); err != nil {
		return err
	}

	return encoder.Decode(bba, enc, &sl.acceptVoteproof)
}
