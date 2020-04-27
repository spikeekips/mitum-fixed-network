package base

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/spikeekips/mitum/util"
)

type ThresholdRLPPacker struct {
	TT uint
	TH uint
	PC []byte
}

func (thr Threshold) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, ThresholdRLPPacker{
		thr.Total,
		thr.Threshold,
		util.Float64ToBytes(thr.Percent),
	})
}

func (thr *Threshold) DecodeRLP(s *rlp.Stream) error {
	var uthr ThresholdRLPPacker
	if err := s.Decode(&uthr); err != nil {
		return err
	}

	thr.Total = uthr.TT
	thr.Threshold = uthr.TH
	thr.Percent = util.BytesToFloat64(uthr.PC)

	return nil
}
