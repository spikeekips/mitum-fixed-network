package block

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testBaseBlockdataMap struct {
	suite.Suite
}

func (t *testBaseBlockdataMap) TestNew() {
	bd := NewBaseBlockdataMap(TestBlockdataWriterHint, 33)
	t.Implements((*BlockdataMap)(nil), bd)
}

func (t *testBaseBlockdataMap) TestRemoteMixed() {
	bd := NewBaseBlockdataMap(TestBlockdataWriterHint, 33)
	bd = bd.SetBlock(valuehash.RandomSHA256())

	for i := range Blockdata {
		item := NewBaseBlockdataMapItem(Blockdata[i], util.UUID().String(), "file:///"+util.UUID().String())
		j, err := bd.SetItem(item)
		t.NoError(err)
		bd = j
	}

	i, err := bd.UpdateHash()
	t.NoError(err)
	bd = i

	t.NoError(bd.IsValid(nil))

	// NOTE set remote item
	item := NewBaseBlockdataMapItem(BlockdataManifest, util.UUID().String(), "https:///"+util.UUID().String())
	j, err := bd.SetItem(item)
	t.NoError(err)
	bd = j

	err = bd.IsValid(nil)
	t.Contains(err.Error(), "all the items should be local or non-local")
}

func TestBaseBlockdataMap(t *testing.T) {
	suite.Run(t, new(testBaseBlockdataMap))
}
