package block

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testBaseBlockDataMap struct {
	suite.Suite
}

func (t *testBaseBlockDataMap) TestNew() {
	bd := NewBaseBlockDataMap(TestBlockDataWriterHint, 33)
	t.Implements((*BlockDataMap)(nil), bd)
}

func (t *testBaseBlockDataMap) TestRemoteMixed() {
	bd := NewBaseBlockDataMap(TestBlockDataWriterHint, 33)
	bd = bd.SetBlock(valuehash.RandomSHA256())

	for i := range BlockData {
		item := NewBaseBlockDataMapItem(BlockData[i], util.UUID().String(), "file:///"+util.UUID().String())
		j, err := bd.SetItem(item)
		t.NoError(err)
		bd = j
	}

	i, err := bd.UpdateHash()
	t.NoError(err)
	bd = i

	t.NoError(bd.IsValid(nil))

	// NOTE set remote item
	item := NewBaseBlockDataMapItem(BlockDataManifest, util.UUID().String(), "https:///"+util.UUID().String())
	j, err := bd.SetItem(item)
	t.NoError(err)
	bd = j

	err = bd.IsValid(nil)
	t.Contains(err.Error(), "all the items should be local or non-local")
}

func TestBaseBlockDataMap(t *testing.T) {
	suite.Run(t, new(testBaseBlockDataMap))
}
