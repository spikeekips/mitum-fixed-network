package process

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type RoundrobinSuffrage struct {
	*BaseSuffrage
}

func NewRoundrobinSuffrage(local *isaac.Local, cacheSize int, numberOfActing uint) *RoundrobinSuffrage {
	sf := &RoundrobinSuffrage{}
	sf.BaseSuffrage = NewBaseSuffrage(
		"roundrobin-suffrage",
		local,
		cacheSize,
		numberOfActing,
		sf.elect,
	)

	return sf
}

func (sf *RoundrobinSuffrage) elect(height base.Height, round base.Round) base.ActingSuffrage {
	all := sf.Nodes()

	if len(all) > 1 {
		sort.Slice(all, func(i, j int) bool {
			return strings.Compare(all[i].String(), all[j].String()) < 0
		})
	}

	na := int(sf.numberOfActing)
	if len(all) < na {
		na = len(all)
	}

	pos := sf.pos(height, round, len(all))

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

	return base.NewActingSuffrage(height, round, all[pos], selected)
}

func (sf *RoundrobinSuffrage) pos(height base.Height, round base.Round, all int) int {
	sum := uint64(height.Int64()) + round.Uint64()

	return int(sum % uint64(all))
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
