package process

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type RoundrobinSuffrage struct {
	*BaseSuffrage
	getManifestFunc func(base.Height) (valuehash.Hash, error)
}

func NewRoundrobinSuffrage(
	nodes []base.Address,
	numberOfActing uint,
	cacheSize int,
	getManifestFunc func(base.Height) (valuehash.Hash, error),
) (*RoundrobinSuffrage, error) {
	sf := &RoundrobinSuffrage{getManifestFunc: getManifestFunc}

	b, err := NewBaseSuffrage(
		"roundrobin-suffrage",
		nodes,
		numberOfActing,
		sf.elect,
		cacheSize,
	)
	if err != nil {
		return nil, err
	}
	sf.BaseSuffrage = b

	return sf, nil
}

func (sf *RoundrobinSuffrage) elect(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	nodes := sf.Nodes()

	na := int(sf.numberOfActing)
	if len(nodes) < na {
		na = len(nodes)
	}

	var proposer base.Address
	var pos int
	if h := height - 1; h <= base.PreGenesisHeight {
		proposer = nodes[0]
	} else if i, err := sf.pos(height, round, len(nodes)); err != nil {
		return base.ActingSuffrage{}, err
	} else {
		pos = i
		proposer = nodes[i]
	}

	var selected []base.Address
	if len(nodes) == na {
		selected = nodes
	} else {
		selected = append(selected, nodes[pos:]...)
		if len(selected) > na {
			selected = selected[:na]
		} else if len(selected) < na {
			selected = append(selected, nodes[:na-len(selected)]...)
		}
	}

	return base.NewActingSuffrage(height, round, proposer, selected), nil
}

func (sf *RoundrobinSuffrage) pos(height base.Height, round base.Round, all int) (int, error) {
	var sum uint64

	// NOTE get manifest of previous height
	if sf.getManifestFunc != nil {
		switch h, err := sf.getManifestFunc(height - 1); {
		case err != nil:
			return 0, err
		default:
			for _, b := range h.Bytes() {
				sum += uint64(b)
			}
		}
	}

	sum += uint64(height.Int64()) + round.Uint64()

	return int(sum % uint64(all)), nil
}

func (sf *RoundrobinSuffrage) Verbose() string {
	m := map[string]interface{}{
		"type":             sf.Name(),
		"cache_size":       sf.CacheSize(),
		"number_of_acting": sf.NumberOfActing(),
	}

	b, err := jsonenc.Marshal(m)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			errors.Wrap(err, "failed to marshal RoundrobinSuffrage.Verbose()").Error(),
		)

		return sf.Name()
	}
	return string(b)
}
