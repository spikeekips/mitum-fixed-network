package base

import (
	"fmt"
	"os"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

// FixedSuffrage will be used for creating genesis block or testing.
type FixedSuffrage struct {
	*logging.Logging
	proposer Address
	nodes    map[Address]struct{}
	nodeList []Address
}

func NewFixedSuffrage(proposer Address, nodes []Address) *FixedSuffrage {
	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer: proposer,
		nodeList: nodes,
	}
}

func (ff *FixedSuffrage) Initialize() error {
	ns := map[Address]struct{}{
		ff.proposer: {},
	}
	nodeList := []Address{ff.proposer}
	for _, n := range ff.nodeList {
		if _, found := ns[n]; found {
			continue
		}

		ns[n] = struct{}{}
		nodeList = append(nodeList, n)
	}

	ff.nodes = ns
	ff.nodeList = nodeList

	return nil
}

func (ff *FixedSuffrage) Name() string {
	return "base-fixed-suffrage"
}

func (ff *FixedSuffrage) IsInside(a Address) bool {
	_, found := ff.nodes[a]
	return found
}

func (ff *FixedSuffrage) Acting(height Height, round Round) (ActingSuffrage, error) {
	return NewActingSuffrage(height, round, ff.proposer, ff.nodeList), nil
}

func (ff *FixedSuffrage) IsActing(_ Height, _ Round, node Address) (bool, error) {
	_, found := ff.nodes[node]
	return found, nil
}

func (ff *FixedSuffrage) IsProposer(_ Height, _ Round, node Address) (bool, error) {
	return ff.proposer.Equal(node), nil
}

func (ff *FixedSuffrage) Nodes() []Address {
	return ff.nodeList
}

func (ff *FixedSuffrage) Verbose() string {
	m := map[string]interface{}{
		"type":     ff.Name(),
		"proposer": ff.proposer,
	}
	if len(ff.Nodes()) > 0 {
		m["nodes"] = ff.Nodes()
	}

	if b, err := jsonenc.Marshal(m); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", xerrors.Errorf("failed to marshal FixedSuffrage.Verbose(): %w", err).Error())

		return ff.Name()
	} else {
		return string(b)
	}
}
