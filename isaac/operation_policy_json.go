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

func (spo SetPolicyOperationV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H                                valuehash.Hash `json:"hash"`
		FH                               valuehash.Hash `json:"fact_hash"`
		FS                               key.Signature  `json:"fact_signature"`
		SN                               key.Publickey  `json:"signer"`
		TK                               []byte         `json:"token"`
		Threshold                        [2]float64     `json:"threshold"`
		TimeoutWaitingProposal           time.Duration  `json:"timeout_waiting_proposal"`
		IntervalBroadcastingINITBallot   time.Duration  `json:"interval_broadcasting_init_ballot"`
		IntervalBroadcastingProposal     time.Duration  `json:"interval_broadcasting_proposal"`
		WaitBroadcastingACCEPTBallot     time.Duration  `json:"wait_broadcasting_accept_ballot"`
		IntervalBroadcastingACCEPTBallot time.Duration  `json:"interval_broadcasting_accept_ballot"`
		NumberOfActingSuffrageNodes      uint           `json:"number_of_acting_suffrage_nodes"`
		TimespanValidBallot              time.Duration  `json:"timespan_valid_ballot"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(spo.Hint()),
		H:                  spo.h,
		FH:                 spo.factHash,
		FS:                 spo.factSignature,
		SN:                 spo.signer,
		TK:                 spo.token,
		Threshold: [2]float64{
			float64(spo.Threshold.Total),
			spo.Threshold.Percent,
		},
		TimeoutWaitingProposal:           spo.TimeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   spo.IntervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     spo.IntervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     spo.WaitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: spo.IntervalBroadcastingACCEPTBallot,
		NumberOfActingSuffrageNodes:      spo.NumberOfActingSuffrageNodes,
		TimespanValidBallot:              spo.TimespanValidBallot,
	})
}

type SetPolicyOperationV0Unpacker struct {
	H                                json.RawMessage `json:"hash"`
	FH                               json.RawMessage `json:"fact_hash"`
	FS                               key.Signature   `json:"fact_signature"`
	SN                               json.RawMessage `json:"signer"`
	TK                               []byte          `json:"token"`
	Threshold                        []float64       `json:"threshold"`
	TimeoutWaitingProposal           time.Duration   `json:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration   `json:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration   `json:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration   `json:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration   `json:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint            `json:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration   `json:"timespan_valid_ballot"`
}

func (spo *SetPolicyOperationV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var usp SetPolicyOperationV0Unpacker
	if err := enc.Unmarshal(b, &usp); err != nil {
		return err
	}

	var err error

	var h, factHash valuehash.Hash
	if h, err = decodeHash(enc, usp.H); err != nil {
		return err
	}
	if factHash, err = decodeHash(enc, usp.FH); err != nil {
		return err
	}
	var signer key.Publickey
	if signer, err = decodePublickey(enc, usp.SN); err != nil {
		return err
	}

	var threshold Threshold
	if len(usp.Threshold) != 2 {
		return xerrors.Errorf("invalid formatted Threshold found: %v", usp.Threshold)
	} else if total := usp.Threshold[0]; total < 0 {
		return xerrors.Errorf("invalid total number of Threshold found: %v", usp.Threshold)
	} else if percent := usp.Threshold[1]; percent < 0 {
		return xerrors.Errorf("invalid percent number of Threshold found: %v", usp.Threshold)
	} else if threshold, err = NewThreshold(uint(total), percent); err != nil {
		return err
	}

	spo.h = h
	spo.factHash = factHash
	spo.factSignature = usp.FS
	spo.SetPolicyOperationFactV0 = SetPolicyOperationFactV0{
		signer:                           signer,
		token:                            usp.TK,
		Threshold:                        threshold,
		TimeoutWaitingProposal:           usp.TimeoutWaitingProposal,
		IntervalBroadcastingINITBallot:   usp.IntervalBroadcastingINITBallot,
		IntervalBroadcastingProposal:     usp.IntervalBroadcastingProposal,
		WaitBroadcastingACCEPTBallot:     usp.WaitBroadcastingACCEPTBallot,
		IntervalBroadcastingACCEPTBallot: usp.IntervalBroadcastingACCEPTBallot,
		NumberOfActingSuffrageNodes:      usp.NumberOfActingSuffrageNodes,
		TimespanValidBallot:              usp.TimespanValidBallot,
	}

	return nil
}
