package base

import (
	"encoding/json"
	"fmt"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type BaseBallotFactSignJSONPacker struct {
	jsonenc.HintedHead
	*BaseFactSignJSONPacker
	*BaseBallotFactSignNodeJSONPacker
}

type BaseBallotFactSignNodeJSONPacker struct {
	NO Address `json:"node"`
}

func (fs BaseBallotFactSign) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseBallotFactSignJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fs.Hint()),
		BaseFactSignJSONPacker: &BaseFactSignJSONPacker{
			SN: fs.Signer(),
			SG: fs.Signature(),
			SA: localtime.NewTime(fs.SignedAt()),
		},
		BaseBallotFactSignNodeJSONPacker: &BaseBallotFactSignNodeJSONPacker{
			NO: fs.node,
		},
	})
}

type BaseBallotFactSignNodeJSONUnpacker struct {
	NO AddressDecoder `json:"node"`
}

func (fs *BaseBallotFactSign) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var bfs BaseFactSign
	if err := bfs.UnpackJSON(b, enc); err != nil {
		return fmt.Errorf("failed to unpack ballot factsign: %w", err)
	}

	var bn BaseBallotFactSignNodeJSONUnpacker
	if err := enc.Unmarshal(b, &bn); err != nil {
		return fmt.Errorf("failed to unpack ballot fact sign: %w", err)
	}

	return fs.unpack(enc, bfs, bn.NO)
}

type BaseSignedBallotFactPackerJSON struct {
	jsonenc.HintedHead
	FC BallotFact `json:"fact"`
	FS FactSign   `json:"fact_sign"`
}

func (sfs BaseSignedBallotFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseSignedBallotFactPackerJSON{
		HintedHead: jsonenc.NewHintedHead(sfs.Hint()),
		FC:         sfs.fact,
		FS:         sfs.factSign,
	})
}

type BaseSignedBallotFactUnpackJSON struct {
	FC json.RawMessage `json:"fact"`
	FS json.RawMessage `json:"fact_sign"`
}

func (sfs *BaseSignedBallotFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var vpp BaseSignedBallotFactUnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return sfs.unpack(enc, vpp.FC, vpp.FS)
}
