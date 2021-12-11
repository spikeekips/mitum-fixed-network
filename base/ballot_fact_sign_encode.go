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
	if err := encoder.Decode(bfc, enc, &sfs.fact); err != nil {
		return err
	}

	return encoder.Decode(bfs, enc, &sfs.factSign)
}
