package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BlockdataMap interface {
	hint.Hinter
	valuehash.HashGenerator
	valuehash.Hasher
	isvalid.IsValider
	// Writer indicates which writer stores block data
	Writer() hint.Hint
	Height() base.Height
	CreatedAt() time.Time
	IsLocal() bool
	Block() valuehash.Hash
	Manifest() BlockdataMapItem
	Operations() BlockdataMapItem
	OperationsTree() BlockdataMapItem
	States() BlockdataMapItem
	StatesTree() BlockdataMapItem
	INITVoteproof() BlockdataMapItem
	ACCEPTVoteproof() BlockdataMapItem
	SuffrageInfo() BlockdataMapItem
	Proposal() BlockdataMapItem
}

type BlockdataMapItem interface {
	isvalid.IsValider
	util.Byter
	Type() string
	Checksum() string
	URL() string
	Exists(string) error
}
