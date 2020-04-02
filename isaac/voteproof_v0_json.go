package isaac

import (
	"encoding/json"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type VoteproofV0PackJSON struct {
	encoder.JSONPackHintedHead
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	TH Threshold          `json:"threshold"`
	RS VoteResultType     `json:"result"`
	ST Stage              `json:"stage"`
	MJ operation.Fact     `json:"majority"`
	FS [][2]interface{}   `json:"facts"`
	BS [][2]interface{}   `json:"ballots"`
	VS [][2]interface{}   `json:"votes"`
	FA localtime.JSONTime `json:"finished_at"`
	CL string             `json:"is_closed"`
}

func (vp VoteproofV0) MarshalJSON() ([]byte, error) {
	var facts [][2]interface{} // nolint
	for h, f := range vp.facts {
		facts = append(facts, [2]interface{}{h, f})
	}

	var ballots [][2]interface{} // nolint
	for a, h := range vp.ballots {
		ballots = append(ballots, [2]interface{}{a, h})
	}

	var votes [][2]interface{} // nolint
	for a := range vp.votes {
		votes = append(votes, [2]interface{}{a, vp.votes[a]})
	}

	var isClosed string
	if vp.closed {
		isClosed = "true"
	} else {
		isClosed = "false"
	}

	return util.JSONMarshal(VoteproofV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(vp.Hint()),
		HT:                 vp.height,
		RD:                 vp.round,
		TH:                 vp.threshold,
		RS:                 vp.result,
		ST:                 vp.stage,
		MJ:                 vp.majority,
		FS:                 facts,
		BS:                 ballots,
		VS:                 votes,
		FA:                 localtime.NewJSONTime(vp.finishedAt),
		CL:                 isClosed,
	})
}

type VoteproofV0UnpackJSON struct {
	HT Height               `json:"height"`
	RD Round                `json:"round"`
	TH Threshold            `json:"threshold"`
	RS VoteResultType       `json:"result"`
	ST Stage                `json:"stage"`
	MJ json.RawMessage      `json:"majority"`
	FS [][2]json.RawMessage `json:"facts"`
	BS [][2]json.RawMessage `json:"ballots"`
	VS [][2]json.RawMessage `json:"votes"`
	FA localtime.JSONTime   `json:"finished_at"`
	CL string               `json:"is_closed"`
}

func (vp *VoteproofV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var vpp VoteproofV0UnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	var err error
	var majority operation.Fact
	if vpp.MJ != nil {
		if majority, err = operation.DecodeFact(enc, vpp.MJ); err != nil {
			return err
		}
	}

	facts := map[valuehash.Hash]operation.Fact{}
	for i := range vpp.FS {
		l := vpp.FS[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of facts; not [2]json.RawMessage")
		}

		var factHash valuehash.Hash
		if factHash, err = valuehash.Decode(enc, l[0]); err != nil {
			return err
		}

		var fact operation.Fact
		if fact, err = operation.DecodeFact(enc, l[1]); err != nil {
			return err
		}

		facts[factHash] = fact
	}

	ballots := map[Address]valuehash.Hash{}
	for i := range vpp.BS {
		l := vpp.BS[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of ballots; not [2]json.RawMessage")
		}

		var address Address
		if address, err = DecodeAddress(enc, l[0]); err != nil {
			return err
		}

		var ballot valuehash.Hash
		if ballot, err = valuehash.Decode(enc, l[1]); err != nil {
			return err
		}

		ballots[address] = ballot
	}

	votes := map[Address]VoteproofNodeFact{}
	for i := range vpp.VS {
		l := vpp.VS[i]
		if len(l) != 2 {
			return xerrors.Errorf("invalid raw of votes; not [2]json.RawMessage")
		}

		var address Address
		if address, err = DecodeAddress(enc, l[0]); err != nil {
			return err
		}

		var nodeFact VoteproofNodeFact
		if err = enc.Decode(l[1], &nodeFact); err != nil {
			return err
		}

		votes[address] = nodeFact
	}

	vp.height = vpp.HT
	vp.round = vpp.RD
	vp.threshold = vpp.TH
	vp.result = vpp.RS
	vp.stage = vpp.ST
	vp.majority = majority
	vp.facts = facts
	vp.ballots = ballots
	vp.votes = votes
	vp.finishedAt = vpp.FA.Time
	vp.closed = vpp.CL == "true"

	return nil
}

type VoteproofNodeFactPackJSON struct {
	FC valuehash.Hash `json:"fact"`
	FS key.Signature  `json:"fact_signature"`
	SG key.Publickey  `json:"signer"`
}

func (vf VoteproofNodeFact) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(VoteproofNodeFactPackJSON{
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

type VoteproofNodeFactUnpackJSON struct {
	FC json.RawMessage `json:"fact"`
	FS key.Signature   `json:"fact_signature"`
	SG json.RawMessage `json:"signer"`
}

func (vf *VoteproofNodeFact) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var vpp VoteproofNodeFactUnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	var fact valuehash.Hash
	if h, err := valuehash.Decode(enc, vpp.FC); err != nil {
		return err
	} else {
		fact = h
	}

	var signer key.Publickey
	if h, err := key.DecodePublickey(enc, vpp.SG); err != nil {
		return err
	} else {
		signer = h
	}

	vf.fact = fact
	vf.factSignature = vpp.FS
	vf.signer = signer

	return nil
}
