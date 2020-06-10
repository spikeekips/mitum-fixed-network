package quicnetwork

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
)

type HeightsArgs struct {
	Heights []base.Height
}

func NewHeightsArgs(heights []base.Height) HeightsArgs {
	return HeightsArgs{Heights: heights}
}

type HashesArgs struct {
	Hashes []valuehash.Hash
}

func NewHashesArgs(hashes []valuehash.Hash) HashesArgs {
	return HashesArgs{Hashes: hashes}
}
