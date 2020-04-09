package isaac

import (
	"io"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
)

type PolicyOperationBodyV0RLPPacker struct {
	H                                hint.Hint
	Threshold                        base.Threshold
	TimeoutWaitingProposal           uint64
	IntervalBroadcastingINITBallot   uint64
	IntervalBroadcastingProposal     uint64
	WaitBroadcastingACCEPTBallot     uint64
	IntervalBroadcastingACCEPTBallot uint64
	NumberOfActingSuffrageNodes      uint
	TimespanValidBallot              uint64
}

func (po PolicyOperationBodyV0) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, PolicyOperationBodyV0RLPPacker{
		H:                                po.Hint(),
		Threshold:                        po.Threshold,
		TimeoutWaitingProposal:           uint64(po.TimeoutWaitingProposal),
		IntervalBroadcastingINITBallot:   uint64(po.IntervalBroadcastingINITBallot),
		IntervalBroadcastingProposal:     uint64(po.IntervalBroadcastingProposal),
		WaitBroadcastingACCEPTBallot:     uint64(po.WaitBroadcastingACCEPTBallot),
		IntervalBroadcastingACCEPTBallot: uint64(po.IntervalBroadcastingACCEPTBallot),
		NumberOfActingSuffrageNodes:      po.NumberOfActingSuffrageNodes,
		TimespanValidBallot:              uint64(po.TimespanValidBallot),
	})
}

func (po *PolicyOperationBodyV0) DecodeRLP(s *rlp.Stream) error {
	var upo PolicyOperationBodyV0RLPPacker
	if err := s.Decode(&upo); err != nil {
		return err
	}

	po.Threshold = upo.Threshold
	po.TimeoutWaitingProposal = time.Duration(upo.TimeoutWaitingProposal)
	po.IntervalBroadcastingINITBallot = time.Duration(upo.IntervalBroadcastingINITBallot)
	po.IntervalBroadcastingProposal = time.Duration(upo.IntervalBroadcastingProposal)
	po.WaitBroadcastingACCEPTBallot = time.Duration(upo.WaitBroadcastingACCEPTBallot)
	po.IntervalBroadcastingACCEPTBallot = time.Duration(upo.IntervalBroadcastingACCEPTBallot)
	po.NumberOfActingSuffrageNodes = upo.NumberOfActingSuffrageNodes
	po.TimespanValidBallot = time.Duration(upo.TimespanValidBallot)

	return nil
}
