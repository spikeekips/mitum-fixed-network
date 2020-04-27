package isaac

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type PolicyOperationBodyV0PackerJSON struct {
	jsonencoder.HintedHead
	Threshold                        []float64     `json:"threshold"`
	TimeoutWaitingProposal           time.Duration `json:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration `json:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration `json:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration `json:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration `json:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint          `json:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration `json:"timespan_valid_ballot"`
}

func (po PolicyOperationBodyV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(PolicyOperationBodyV0PackerJSON{
		HintedHead: jsonencoder.NewHintedHead(po.Hint()),
		Threshold: []float64{
			float64(po.Threshold.Total),
			po.Threshold.Percent,
		},
		TimeoutWaitingProposal:           po.TimeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   po.IntervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     po.IntervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     po.WaitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: po.IntervalBroadcastingACCEPTBallot,
		NumberOfActingSuffrageNodes:      po.NumberOfActingSuffrageNodes,
		TimespanValidBallot:              po.TimespanValidBallot,
	})
}

type PolicyOperationBodyV0UnpackerJSON struct {
	PolicyOperationBodyV0PackerJSON
}

func (po *PolicyOperationBodyV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var up PolicyOperationBodyV0UnpackerJSON
	if err := enc.Unmarshal(b, &up); err != nil {
		return err
	}

	return po.unpack(
		up.Threshold,
		up.TimeoutWaitingProposal,
		up.IntervalBroadcastingINITBallot,
		up.IntervalBroadcastingProposal,
		up.WaitBroadcastingACCEPTBallot,
		up.IntervalBroadcastingACCEPTBallot,
		up.NumberOfActingSuffrageNodes,
		up.TimespanValidBallot,
	)
}

func (spo SetPolicyOperationV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		H  valuehash.Hash        `json:"hash"`
		FH valuehash.Hash        `json:"fact_hash"`
		FS key.Signature         `json:"fact_signature"`
		SN key.Publickey         `json:"signer"`
		TK []byte                `json:"token"`
		PO PolicyOperationBodyV0 `json:"policies"`
	}{
		HintedHead: jsonencoder.NewHintedHead(spo.Hint()),
		H:          spo.h,
		FH:         spo.factHash,
		FS:         spo.factSignature,
		SN:         spo.signer,
		TK:         spo.token,
		PO:         spo.SetPolicyOperationFactV0.PolicyOperationBodyV0,
	})
}

type SetPolicyOperationV0UnpackerJSON struct {
	H  json.RawMessage `json:"hash"`
	FH json.RawMessage `json:"fact_hash"`
	FS key.Signature   `json:"fact_signature"`
	SN json.RawMessage `json:"signer"`
	TK []byte          `json:"token"`
	PO json.RawMessage `json:"policies"`
}

func (spo *SetPolicyOperationV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var usp SetPolicyOperationV0UnpackerJSON
	if err := enc.Unmarshal(b, &usp); err != nil {
		return err
	}

	return spo.unpack(enc, usp.H, usp.FH, usp.FS, usp.SN, usp.TK, usp.PO)
}
