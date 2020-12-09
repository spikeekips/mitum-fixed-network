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

type FixedSuffrage struct {
	*BaseSuffrage
	proposer base.Address
	acting   []base.Address
}

func NewFixedSuffrage(
	local *isaac.Local,
	cacheSize int,
	proposer base.Address,
	acting []base.Address,
) (*FixedSuffrage, error) {
	if proposer == nil && len(acting) < 1 {
		return nil, xerrors.Errorf("empty proposer and nodes")
	}

	sf := &FixedSuffrage{proposer: proposer}

	var elect ActinfSuffrageElectFunc
	if proposer == nil {
		if len(acting) == 1 {
			sf.proposer = acting[0]
			elect = sf.electWithProposer
		} else {
			elect = sf.elect
		}
	} else {
		var found bool
		for i := range acting {
			if acting[i].Equal(proposer) {
				found = true

				break
			}
		}

		if !found {
			acting = append(acting, proposer)
		}

		sort.Slice(acting, func(i, j int) bool {
			return strings.Compare(acting[i].String(), acting[j].String()) < 0
		})

		elect = sf.electWithProposer
	}

	sf.acting = acting

	sf.BaseSuffrage = NewBaseSuffrage(
		"fixed-suffrage",
		local,
		cacheSize,
		uint(len(acting)),
		elect,
	)

	return sf, nil
}

func (sf *FixedSuffrage) electWithProposer(height base.Height, round base.Round) base.ActingSuffrage {
	return base.NewActingSuffrage(height, round, sf.proposer, sf.acting)
}

func (sf *FixedSuffrage) elect(height base.Height, round base.Round) base.ActingSuffrage {
	pos := (uint64(height) + round.Uint64()) % uint64(len(sf.acting))

	return base.NewActingSuffrage(height, round, sf.acting[pos], sf.acting)
}

func (sf *FixedSuffrage) Verbose() string {
	m := map[string]interface{}{
		"type":             sf.Name(),
		"cache_size":       sf.CacheSize(),
		"number_of_acting": sf.NumberOfActing(),
		"proposer":         sf.proposer,
		"acting":           sf.acting,
	}

	if b, err := jsonenc.Marshal(m); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			xerrors.Errorf("failed to marshal FixedSuffrage.Verbose(): %w", err).Error(),
		)

		return sf.Name()
	} else {
		return string(b)
	}
}
