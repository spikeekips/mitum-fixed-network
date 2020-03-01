// +build test

package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
)

var TestNetworkID []byte = []byte(fmt.Sprintf("network-id-%s", util.UUID().String()))
