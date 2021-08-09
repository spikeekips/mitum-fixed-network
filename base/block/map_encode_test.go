package block

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testBlockDataMapEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testBlockDataMapEncode) SetupSuite() {
	t.enc.Add(BaseBlockDataMap{})
}

func (t *testBlockDataMapEncode) TestMarshal() {
	bd := NewBaseBlockDataMap(TestBlockDataWriterHint, 33)
	bd = bd.SetBlock(valuehash.RandomSHA256())

	u := func() string {
		return "file:///" + util.UUID().String()
	}

	var item BaseBlockDataMapItem
	item = NewBaseBlockDataMapItem(BlockDataManifest, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataOperations, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataOperationsTree, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataStates, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataStatesTree, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataINITVoteproof, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataACCEPTVoteproof, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataSuffrageInfo, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockDataMapItem(BlockDataProposal, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)

	i, err := bd.UpdateHash()
	t.NoError(err)
	bd = i

	t.NoError(bd.IsValid(nil))
	t.True(bd.IsLocal())

	b, err := t.enc.Marshal(bd)
	t.NoError(err)

	j, err := t.enc.Decode(b)
	t.NoError(err)

	t.IsType(BaseBlockDataMap{}, j)

	ubd := j.(BaseBlockDataMap)

	t.True(bd.h.Equal(ubd.h))
	t.True(bd.writerHint.Equal(ubd.writerHint))
	t.Equal(bd.height, ubd.height)
	t.True(bd.block.Equal(ubd.block))
	t.True(localtime.Equal(bd.CreatedAt(), ubd.CreatedAt()))
	for k := range bd.items {
		t.Equal(bd.items[k], ubd.items[k])
	}
}

func TestBlockDataMapEncodeJSON(t *testing.T) {
	b := new(testBlockDataMapEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBlockDataMapEncodeBSON(t *testing.T) {
	b := new(testBlockDataMapEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
