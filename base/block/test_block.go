// +build test

package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/assert"
)

var (
	TestBlockDataWriterType = hint.Type("test-blockdata-writer")
	TestBlockDataWriterHint = hint.NewHint(TestBlockDataWriterType, "v0.0.1")
)

func NewTestBlockV0(height base.Height, round base.Round, proposal, previousBlock valuehash.Hash) (BlockV0, error) {
	nodes := []base.Node{base.RandomNode(util.UUID().String())}

	return NewBlockV0(
		NewSuffrageInfoV0(nodes[0].Address(), nodes),
		height,
		round,
		proposal,
		previousBlock,
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.UTCNow(),
	)
}

func CompareBlockDataMap(t *assert.Assertions, a, b BlockDataMap) {
	t.True(a.Hash().Equal(b.Hash()))
	t.Equal(a.Height(), b.Height())
	t.True(a.Block().Equal(b.Block()))
	t.True(localtime.Equal(a.CreatedAt(), b.CreatedAt()))

	CompareBlockDataMapItem(t, a.Manifest(), b.Manifest())
	CompareBlockDataMapItem(t, a.Operations(), b.Operations())
	CompareBlockDataMapItem(t, a.OperationsTree(), b.OperationsTree())
	CompareBlockDataMapItem(t, a.States(), b.States())
	CompareBlockDataMapItem(t, a.StatesTree(), b.StatesTree())
	CompareBlockDataMapItem(t, a.INITVoteproof(), b.INITVoteproof())
	CompareBlockDataMapItem(t, a.ACCEPTVoteproof(), b.ACCEPTVoteproof())
	CompareBlockDataMapItem(t, a.SuffrageInfo(), b.SuffrageInfo())
	CompareBlockDataMapItem(t, a.Proposal(), b.Proposal())
}

func CompareBlockDataMapItem(t *assert.Assertions, a, b BlockDataMapItem) {
	t.Equal(a.Type(), b.Type())
	t.Equal(a.Checksum(), b.Checksum())
	t.Equal(a.URL(), b.URL())
}

func (bd BaseBlockDataMap) SetHash(h valuehash.Hash) BaseBlockDataMap {
	bd.h = h

	return bd
}
