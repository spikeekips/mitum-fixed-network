package base

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

const AddressTypeSize = 3

// Address represents the address of account.
type Address interface {
	fmt.Stringer // NOTE String() should be typed string
	isvalid.IsValider
	hint.Hinter
	util.Byter
	Equal(Address) bool
}

func SortAddresses(as []Address) {
	sort.Slice(as, func(i, j int) bool {
		return strings.Compare(
			as[i].String(),
			as[j].String(),
		) < 0
	})
}
