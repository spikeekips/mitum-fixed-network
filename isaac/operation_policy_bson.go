package isaac

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (po PolicyOperationBodyV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(po.Hint()),
		bson.M{
			"threshold":                           po.thresholdRatio,
			"timeout_waiting_proposal":            po.timeoutWaitingProposal,
			"interval_broadcasting_init_ballot":   po.intervalBroadcastingINITBallot,
			"interval_broadcasting_proposal":      po.intervalBroadcastingProposal,
			"wait_broadcasting_accept_ballot":     po.waitBroadcastingACCEPTBallot,
			"interval_broadcasting_accept_ballot": po.intervalBroadcastingACCEPTBallot,
			"number_of_acting_suffrage_nodes":     po.numberOfActingSuffrageNodes,
			"timespan_valid_ballot":               po.timespanValidBallot,
			"timeout_process_proposal":            po.timeoutProcessProposal,
		},
	))
}

type PolicyOperationBodyV0UnpackerBSON struct {
	ThresholdRatio                   base.ThresholdRatio `bson:"threshold"`
	TimeoutWaitingProposal           time.Duration       `bson:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration       `bson:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration       `bson:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration       `bson:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration       `bson:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint                `bson:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration       `bson:"timespan_valid_ballot"`
	TimeoutProcessProposal           time.Duration       `bson:"timeout_process_proposal"`
}

func (po *PolicyOperationBodyV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var up PolicyOperationBodyV0UnpackerBSON
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

func (spo SetPolicyOperationV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(spo.Hint()),
		bson.M{
			"hash":           spo.h,
			"fact_hash":      spo.factHash,
			"fact_signature": spo.factSignature,
			"signer":         spo.signer,
			"token":          spo.token,
			"policies":       spo.SetPolicyOperationFactV0.PolicyOperationBodyV0,
		},
	))
}

type SetPolicyOperationV0UnpackerBSON struct {
	H  valuehash.Bytes `bson:"hash"`
	FH valuehash.Bytes `bson:"fact_hash"`
	FS key.Signature   `bson:"fact_signature"`
	SN key.KeyDecoder  `bson:"signer"`
	TK []byte          `bson:"token"`
	PO bson.Raw        `bson:"policies"`
}

func (spo *SetPolicyOperationV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var usp SetPolicyOperationV0UnpackerBSON
	if err := enc.Unmarshal(b, &usp); err != nil {
		return err
	}

	return spo.unpack(enc, usp.H, usp.FH, usp.FS, usp.SN, usp.TK, usp.PO)
}
