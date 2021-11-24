package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (sl *BaseSeal) unpack(
	enc encoder.Encoder,
	bf,
	bbb,
	bba []byte,
) error {
	sfs, err := base.DecodeSignedBallotFact(bf, enc)
	if err != nil {
		return err
	}

	var bb, ba base.Voteproof
	if len(bbb) > 0 {
		i, err := base.DecodeVoteproof(bbb, enc)
		if err != nil {
			return err
		}

		bb = i
	}

	if len(bba) > 0 {
		i, err := base.DecodeVoteproof(bba, enc)
		if err != nil {
			return err
		}

		ba = i
	}

	sl.sfs = sfs
	sl.baseVoteproof = bb
	sl.acceptVoteproof = ba

	return nil
}
