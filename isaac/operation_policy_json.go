package isaac

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type PolicyOperationBodyV0PackerJSON struct {
	encoder.JSONPackHintedHead
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
	return util.JSONMarshal(PolicyOperationBodyV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(po.Hint()),
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

func (po *PolicyOperationBodyV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var up PolicyOperationBodyV0UnpackerJSON
	if err := enc.Unmarshal(b, &up); err != nil {
		return err
	}

	var err error

	var threshold Threshold
	if len(up.Threshold) != 2 {
		return xerrors.Errorf("invalid formatted Threshold found: %v", up.Threshold)
	} else if total := up.Threshold[0]; total < 0 {
		return xerrors.Errorf("invalid total number of Threshold found: %v", up.Threshold)
	} else if percent := up.Threshold[1]; percent < 0 {
		return xerrors.Errorf("invalid percent number of Threshold found: %v", up.Threshold)
	} else if threshold, err = NewThreshold(uint(total), percent); err != nil {
		return err
	}

	po.Threshold = threshold
	po.TimeoutWaitingProposal = up.TimeoutWaitingProposal
	po.IntervalBroadcastingINITBallot = up.IntervalBroadcastingINITBallot
	po.IntervalBroadcastingProposal = up.IntervalBroadcastingProposal
	po.WaitBroadcastingACCEPTBallot = up.WaitBroadcastingACCEPTBallot
	po.IntervalBroadcastingACCEPTBallot = up.IntervalBroadcastingACCEPTBallot
	po.NumberOfActingSuffrageNodes = up.NumberOfActingSuffrageNodes
	po.TimespanValidBallot = up.TimespanValidBallot

	return nil
}

func (spo SetPolicyOperationV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H  valuehash.Hash        `json:"hash"`
		FH valuehash.Hash        `json:"fact_hash"`
		FS key.Signature         `json:"fact_signature"`
		SN key.Publickey         `json:"signer"`
		TK []byte                `json:"token"`
		PO PolicyOperationBodyV0 `json:"policies"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(spo.Hint()),
		H:                  spo.h,
		FH:                 spo.factHash,
		FS:                 spo.factSignature,
		SN:                 spo.signer,
		TK:                 spo.token,
		PO:                 spo.SetPolicyOperationFactV0.PolicyOperationBodyV0,
	})
}

type SetPolicyOperationV0Unpacker struct {
	H  json.RawMessage `json:"hash"`
	FH json.RawMessage `json:"fact_hash"`
	FS key.Signature   `json:"fact_signature"`
	SN json.RawMessage `json:"signer"`
	TK []byte          `json:"token"`
	PO json.RawMessage `json:"policies"`
}

func (spo *SetPolicyOperationV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var usp SetPolicyOperationV0Unpacker
	if err := enc.Unmarshal(b, &usp); err != nil {
		return err
	}

	var err error

	var h, factHash valuehash.Hash
	if h, err = valuehash.Decode(enc, usp.H); err != nil {
		return err
	}
	if factHash, err = valuehash.Decode(enc, usp.FH); err != nil {
		return err
	}
	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, usp.SN); err != nil {
		return err
	}

	var body PolicyOperationBodyV0
	if err := enc.Decode(usp.PO, &body); err != nil {
		return err
	}

	spo.h = h
	spo.factHash = factHash
	spo.factSignature = usp.FS
	spo.SetPolicyOperationFactV0 = SetPolicyOperationFactV0{
		PolicyOperationBodyV0: body,
		signer:                signer,
		token:                 usp.TK,
	}

	return nil
}
