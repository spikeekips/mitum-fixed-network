package base

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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

func (*FixedSuffrage) Name() string {
	return "base-fixed-suffrage"
}

func (*FixedSuffrage) NumberOfActing() uint {
	return 1
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

	b, err := jsonenc.Marshal(m)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr,
			"%+v\n", errors.Wrap(err, "failed to marshal FixedSuffrage.Verbose()").Error(),
		)

		return ff.Name()
	}

	return string(b)
}
