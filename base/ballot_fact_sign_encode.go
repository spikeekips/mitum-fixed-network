package base

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (fs *BaseBallotFactSign) unpack(enc encoder.Encoder, bfs BaseFactSign, bn AddressDecoder) error {
	n, err := bn.Encode(enc)
	if err != nil {
		return err
	}

	fs.BaseFactSign = bfs
	fs.node = n

	return nil
}

func (sfs *BaseSignedBallotFact) unpack(
	enc encoder.Encoder,
	bfc,
	bfs []byte,
) error {
	fact, err := DecodeBallotFact(bfc, enc)
	if err != nil {
		return err
	}

	fs, err := DecodeBallotFactSign(bfs, enc)
	if err != nil {
		return err
	}

	sfs.fact = fact
	sfs.factSign = fs

	return nil
}
