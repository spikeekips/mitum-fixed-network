package base

import "github.com/spikeekips/mitum/util/logging"

func (as ActingSuffrage) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	ev := logging.Dict().
		HintedVerbose("height", as.height, verbose).
		HintedVerbose("round", as.round, verbose).
		Int("number_of_nodes", len(as.nodes)).
		HintedVerbose("proposer", as.proposer, verbose)

	if !verbose {
		return e.Dict(key, ev)
	}

	nodes := make([]string, len(as.nodes))
	for i, n := range as.nodeList {
		nodes[i] = n.String()
	}

	return e.Dict(key, ev.Strs("nodes", nodes))
}
