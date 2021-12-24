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

type testBlockdataMapEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testBlockdataMapEncode) SetupSuite() {
	t.enc.Add(BaseBlockdataMapHinter)
}

func (t *testBlockdataMapEncode) TestMarshal() {
	bd := NewBaseBlockdataMap(TestBlockdataWriterHint, 33)
	bd = bd.SetBlock(valuehash.RandomSHA256())

	u := func() string {
		return "file:///" + util.UUID().String()
	}

	var item BaseBlockdataMapItem
	item = NewBaseBlockdataMapItem(BlockdataManifest, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataOperations, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataOperationsTree, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataStates, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataStatesTree, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataINITVoteproof, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataACCEPTVoteproof, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataSuffrageInfo, valuehash.RandomSHA256().String(), u())
	bd.SetItem(item)
	item = NewBaseBlockdataMapItem(BlockdataProposal, valuehash.RandomSHA256().String(), u())
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

	t.IsType(BaseBlockdataMap{}, j)

	ubd := j.(BaseBlockdataMap)

	t.True(bd.h.Equal(ubd.h))
	t.True(bd.writerHint.Equal(ubd.writerHint))
	t.Equal(bd.height, ubd.height)
	t.True(bd.block.Equal(ubd.block))
	t.True(localtime.Equal(bd.CreatedAt(), ubd.CreatedAt()))
	for k := range bd.items {
		t.Equal(bd.items[k], ubd.items[k])
	}
}

func TestBlockdataMapEncodeJSON(t *testing.T) {
	b := new(testBlockdataMapEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBlockdataMapEncodeBSON(t *testing.T) {
	b := new(testBlockdataMapEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
