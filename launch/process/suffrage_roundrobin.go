package process

import (
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type RoundrobinSuffrage struct {
	*BaseSuffrage
	getManifestFunc func(base.Height) (valuehash.Hash, error)
}

func NewRoundrobinSuffrage(
	local *isaac.Local,
	cacheSize int,
	numberOfActing uint,
	getManifestFunc func(base.Height) (valuehash.Hash, error),
) *RoundrobinSuffrage {
	sf := &RoundrobinSuffrage{getManifestFunc: getManifestFunc}
	sf.BaseSuffrage = NewBaseSuffrage(
		"roundrobin-suffrage",
		local,
		cacheSize,
		numberOfActing,
		sf.elect,
	)

	return sf
}

func (sf *RoundrobinSuffrage) elect(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	all := sf.Nodes()
	base.SortAddresses(all)

	na := int(sf.numberOfActing)
	if len(all) < na {
		na = len(all)
	}

	var proposer base.Address
	var pos int
	if h := height - 1; h <= base.PreGenesisHeight {
		proposer = sf.local.Node().Address()
	} else if i, err := sf.pos(height, round, len(all)); err != nil {
		return base.ActingSuffrage{}, err
	} else {
		pos = i
		proposer = all[i]
	}

	var selected []base.Address
	if len(all) == na {
		selected = append(selected, all...)
	} else {
		selected = append(selected, all[pos:]...)
		if len(selected) > na {
			selected = selected[:na]
		} else if len(selected) < na {
			selected = append(selected, all[:na-len(selected)]...)
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

	if b, err := jsonenc.Marshal(m); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			xerrors.Errorf("failed to marshal RoundrobinSuffrage.Verbose(): %w", err).Error(),
		)

		return sf.Name()
	} else {
		return string(b)
	}
}
