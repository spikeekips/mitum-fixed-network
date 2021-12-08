//go:build test
// +build test

package base

import "github.com/spikeekips/mitum/util"

func init() {
	MinAddressSize = AddressTypeSize + 2
}

func RandomStringAddress() StringAddress {
	return MustNewStringAddress(util.UUID().String())
}
