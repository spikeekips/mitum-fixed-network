package base

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (as ActingSuffrage) MarshalZerologObject(e *zerolog.Event) {
	e.
		Int64("height", as.height.Int64()).
		Uint64("round", as.round.Uint64()).
		Int("number_of_nodes", len(as.nodes)).
		Stringer("proposer", as.proposer)

	if e := log.Debug(); e.Enabled() {
		nodes := make([]string, len(as.nodes))
		for i, n := range as.nodeList {
			nodes[i] = n.String()
		}

		e.Strs("nodes", nodes)
	}
}
