package isaac

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
)

func (vp VoteProofV0) MarshalZerologObject(e *zerolog.Event) {
	r, _ := util.JSONMarshal(vp)

	e.RawJSON("voteproof", r)
}
