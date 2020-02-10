package isaac

import (
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type VoteProofNodeFactPackJSON struct {
	FC valuehash.Hash `json:"fact"`
	FS key.Signature  `json:"fact_signature"`
	SG key.Publickey  `json:"signer"`
}

type VoteProofV0PackJSON struct {
	HT Height                       `json:"height"`
	RD Round                        `json:"round"`
	TH Threshold                    `json:"threshold"`
	RS VoteProofResultType          `json:"result"`
	ST Stage                        `json:"stage"`
	MJ Fact                         `json:"majority"`
	FS map[string]Fact              `json:"facts"`
	BS map[string]valuehash.Hash    `json:"ballots"`
	VS map[string]VoteProofNodeFact `json:"votes"`
}

func (vp VoteProofV0) MarshalJSON() ([]byte, error) {
	facts := map[string]Fact{}
	for h, f := range vp.facts {
		facts[h.String()] = f
	}

	ballots := map[string]valuehash.Hash{}
	for a, h := range vp.ballots {
		ballots[a.String()] = h
	}

	votes := map[string]VoteProofNodeFact{}
	for a := range vp.votes {
		votes[a.String()] = vp.votes[a]
	}

	return util.JSONMarshal(VoteProofV0PackJSON{
		HT: vp.height,
		RD: vp.round,
		TH: vp.threshold,
		RS: vp.result,
		ST: vp.stage,
		MJ: vp.majority,
		FS: facts,
		BS: ballots,
		VS: votes,
	})
}

func (vf VoteProofNodeFact) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(VoteProofNodeFactPackJSON{
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

// TODO UnpackXXX
