//go:build test
// +build test

package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/assert"
)

var (
	TestBlockdataWriterType = hint.Type("test-blockdata-writer")
	TestBlockdataWriterHint = hint.NewHint(TestBlockdataWriterType, "v0.0.1")
)

func NewTestBlockV0(height base.Height, round base.Round, proposalFact, previousBlock valuehash.Hash) (BlockV0, error) {
	nodes := []base.Node{node.RandomNode(util.UUID().String())}

	return NewBlockV0(
		NewSuffrageInfoV0(nodes[0].Address(), nodes),
		height,
		round,
		proposalFact,
		previousBlock,
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.UTCNow(),
	)
}

func CompareBlockdataMap(t *assert.Assertions, a, b BlockdataMap) {
	t.True(a.Hash().Equal(b.Hash()))
	t.Equal(a.Height(), b.Height())
	t.True(a.Block().Equal(b.Block()))
	t.True(localtime.Equal(a.CreatedAt(), b.CreatedAt()))

	CompareBlockdataMapItem(t, a.Manifest(), b.Manifest())
	CompareBlockdataMapItem(t, a.Operations(), b.Operations())
	CompareBlockdataMapItem(t, a.OperationsTree(), b.OperationsTree())
	CompareBlockdataMapItem(t, a.States(), b.States())
	CompareBlockdataMapItem(t, a.StatesTree(), b.StatesTree())
	CompareBlockdataMapItem(t, a.INITVoteproof(), b.INITVoteproof())
	CompareBlockdataMapItem(t, a.ACCEPTVoteproof(), b.ACCEPTVoteproof())
	CompareBlockdataMapItem(t, a.SuffrageInfo(), b.SuffrageInfo())
	CompareBlockdataMapItem(t, a.Proposal(), b.Proposal())
}

func CompareBlockdataMapItem(t *assert.Assertions, a, b BlockdataMapItem) {
	t.Equal(a.Type(), b.Type())
	t.Equal(a.Checksum(), b.Checksum())
	t.Equal(a.URL(), b.URL())
}

func (bd BaseBlockdataMap) SetHash(h valuehash.Hash) BaseBlockdataMap {
	bd.h = h

	return bd
}
