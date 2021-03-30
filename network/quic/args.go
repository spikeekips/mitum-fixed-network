package quicnetwork

import (
	"bytes"
	"sort"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/valuehash"
)

type HeightsArgs struct {
	Heights []base.Height
}

func NewHeightsArgs(heights []base.Height) HeightsArgs {
	return HeightsArgs{Heights: heights}
}

func (ha HeightsArgs) Sort() {
	l := ha.Heights
	sort.Slice(l, func(i, j int) bool {
		return l[i] < l[j]
	})
}

func (ha HeightsArgs) String() string {
	s := ""
	for i := range ha.Heights {
		s += "." + ha.Heights[i].String()
	}

	return s
}

type HashesArgs struct {
	Hashes []valuehash.Hash
}

func NewHashesArgs(hashes []valuehash.Hash) HashesArgs {
	return HashesArgs{Hashes: hashes}
}

func (ha HashesArgs) Sort() {
	l := ha.Hashes
	sort.Slice(l, func(i, j int) bool {
		return bytes.Compare(l[i].Bytes(), l[j].Bytes()) < 0
	})
}

func (ha HashesArgs) String() string {
	s := ""
	for i := range ha.Hashes {
		s += "." + ha.Hashes[i].String()
	}

	return s
}
