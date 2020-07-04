package isaac

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type PolicyOperationBodyV0PackerJSON struct {
	jsonenc.HintedHead
	ThresholdRatio                   base.ThresholdRatio `json:"threshold"`
	TimeoutWaitingProposal           time.Duration       `json:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration       `json:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration       `json:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration       `json:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration       `json:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint                `json:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration       `json:"timespan_valid_ballot"`
	TimeoutProcessProposal           time.Duration       `json:"timeout_process_proposal"`
}

func (po PolicyOperationBodyV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(PolicyOperationBodyV0PackerJSON{
		HintedHead:                       jsonenc.NewHintedHead(po.Hint()),
		ThresholdRatio:                   po.thresholdRatio,
		TimeoutWaitingProposal:           po.timeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   po.intervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     po.intervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     po.waitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: po.intervalBroadcastingACCEPTBallot,
		NumberOfActingSuffrageNodes:      po.numberOfActingSuffrageNodes,
		TimespanValidBallot:              po.timespanValidBallot,
		TimeoutProcessProposal:           po.timeoutProcessProposal,
	})
}

type PolicyOperationBodyV0UnpackerJSON struct {
	PolicyOperationBodyV0PackerJSON
}

func (po *PolicyOperationBodyV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var up PolicyOperationBodyV0UnpackerJSON
	if err := enc.Unmarshal(b, &up); err != nil {
		return err
	}

	return po.unpack(
		up.ThresholdRatio,
		up.TimeoutWaitingProposal,
		up.IntervalBroadcastingINITBallot,
		up.IntervalBroadcastingProposal,
		up.WaitBroadcastingACCEPTBallot,
		up.IntervalBroadcastingACCEPTBallot,
		up.NumberOfActingSuffrageNodes,
		up.TimespanValidBallot,
		up.TimeoutProcessProposal,
	)
}

func (spo SetPolicyOperationV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H  valuehash.Hash        `json:"hash"`
		FS []operation.FactSign  `json:"fact_signs"`
		TK []byte                `json:"token"`
		PO PolicyOperationBodyV0 `json:"policies"`
	}{
		HintedHead: jsonenc.NewHintedHead(spo.Hint()),
		H:          spo.h,
		FS:         []operation.FactSign{spo.fs},
		TK:         spo.token,
		PO:         spo.SetPolicyOperationFactV0.PolicyOperationBodyV0,
	})
}

type SetPolicyOperationV0UnpackerJSON struct {
	H  valuehash.Bytes   `json:"hash"`
	FS []json.RawMessage `json:"fact_signs"`
	TK []byte            `json:"token"`
	PO json.RawMessage   `json:"policies"`
}

func (spo *SetPolicyOperationV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var usp SetPolicyOperationV0UnpackerJSON
	if err := enc.Unmarshal(b, &usp); err != nil {
		return err
	}

	var fs []byte
	if len(usp.FS) > 0 {
		fs = usp.FS[0]
	}

	return spo.unpack(enc, usp.H, fs, usp.TK, usp.PO)
}
