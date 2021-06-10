package process

import (
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type FixedSuffrage struct {
	*BaseSuffrage
	proposer base.Address
}

func NewFixedSuffrage(
	proposer base.Address,
	nodes []base.Address,
	numberOfActing uint,
	cacheSize int,
) (*FixedSuffrage, error) {
	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes")
	}

	sf := &FixedSuffrage{proposer: proposer}

	var elect ActinfSuffrageElectFunc
	if proposer == nil {
		if len(nodes) == 1 {
			sf.proposer = nodes[0]
			elect = sf.electWithProposer
		} else {
			elect = sf.elect
		}
	} else {
		var found bool
		for i := range nodes {
			if nodes[i].Equal(proposer) {
				found = true

				break
			}
		}

		if !found {
			return nil, xerrors.Errorf("proposer not found in nodes")
		}

		elect = sf.electWithProposer
	}

	b, err := NewBaseSuffrage(
		"fixed-suffrage",
		nodes,
		numberOfActing,
		elect,
		cacheSize,
	)
	if err != nil {
		return nil, err
	}
	sf.BaseSuffrage = b

	return sf, nil
}

func (sf *FixedSuffrage) electWithProposer(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	return base.NewActingSuffrage(height, round, sf.proposer, sf.Nodes()), nil
}

func (sf *FixedSuffrage) elect(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	pos := (uint64(height) + round.Uint64()) % uint64(sf.NumberOfActing())

	nodes := sf.Nodes()
	return base.NewActingSuffrage(height, round, nodes[pos], nodes), nil
}

func (sf *FixedSuffrage) Verbose() string {
	m := map[string]interface{}{
		"type":             sf.Name(),
		"cache_size":       sf.CacheSize(),
		"number_of_acting": sf.NumberOfActing(),
		"proposer":         sf.proposer,
		"nodes":            sf.nodes,
	}

	b, err := jsonenc.Marshal(m)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			xerrors.Errorf("failed to marshal FixedSuffrage.Verbose(): %w", err).Error(),
		)

		return sf.Name()
	}
	return string(b)
}
