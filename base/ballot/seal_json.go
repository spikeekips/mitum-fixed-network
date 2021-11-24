package ballot

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseBallotPackerJSON struct {
	F  base.SignedBallotFact `json:"signed_fact"`
	BB base.Voteproof        `json:"base_voteproof"`
	BA base.Voteproof        `json:"accept_voteproof,omitempty"`
}

func (sl BaseSeal) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		*seal.BaseSealJSONPack
		*BaseBallotPackerJSON
	}{
		BaseSealJSONPack: sl.BaseSeal.JSONPacker(),
		BaseBallotPackerJSON: &BaseBallotPackerJSON{
			F:  sl.sfs,
			BB: sl.baseVoteproof,
			BA: sl.acceptVoteproof,
		},
	})
}

type BaseBallotUnpackerJSON struct {
	F  json.RawMessage `json:"signed_fact"`
	BB json.RawMessage `json:"base_voteproof"`
	BA json.RawMessage `json:"accept_voteproof,omitempty"`
}

func (sl *BaseSeal) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	if err := sl.BaseSeal.UnpackJSON(b, enc); err != nil {
		return err
	}

	var ub BaseBallotUnpackerJSON
	if err := enc.Unmarshal(b, &ub); err != nil {
		return err
	}

	return sl.unpack(enc, ub.F, ub.BB, ub.BA)
}
