package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BlockDataMap interface {
	hint.Hinter
	valuehash.HashGenerator
	util.Byter
	isvalid.IsValider
	// Writer indicates which writer stores block data
	Writer() hint.Hint
	Height() base.Height
	Hash() valuehash.Hash
	CreatedAt() time.Time
	IsLocal() bool
	Block() valuehash.Hash
	Manifest() BlockDataMapItem
	Operations() BlockDataMapItem
	OperationsTree() BlockDataMapItem
	States() BlockDataMapItem
	StatesTree() BlockDataMapItem
	INITVoteproof() BlockDataMapItem
	ACCEPTVoteproof() BlockDataMapItem
	SuffrageInfo() BlockDataMapItem
	Proposal() BlockDataMapItem
}

type BlockDataMapItem interface {
	isvalid.IsValider
	util.Byter
	Type() string
	Checksum() string
	URL() string
	Exists(string) error
}
