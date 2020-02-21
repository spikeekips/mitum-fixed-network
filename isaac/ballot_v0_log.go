package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/util"
)

func (ib INITBallotV0) MarshalZerologObject(e *zerolog.Event) {
	r, _ := util.JSONMarshal(ib)

	e.RawJSON("ballot", r)
}

func (pr ProposalV0) MarshalZerologObject(e *zerolog.Event) {
	r, _ := util.JSONMarshal(pr)

	e.RawJSON("proposal", r)
}

func (sb SIGNBallotV0) MarshalZerologObject(e *zerolog.Event) {
	r, _ := util.JSONMarshal(sb)

	e.RawJSON("ballot", r)
}

func (ab ACCEPTBallotV0) MarshalZerologObject(e *zerolog.Event) {
	r, _ := util.JSONMarshal(ab)

	e.RawJSON("ballot", r)
}
