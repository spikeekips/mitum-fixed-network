// +build mongodb

package mongodbstorage

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testStorage struct {
	storage.BaseTestStorage
	storage *Storage
}

func (t *testStorage) SetupTest() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	st, err := NewStorage(client, t.Encs, t.BSONEnc)
	t.NoError(err)
	t.storage = st
}

func (t *testStorage) TearDownTest() {
	t.storage.Client().DropDatabase()
	t.storage.Close()
}

func (t *testStorage) TestNew() {
	t.Implements((*storage.Storage)(nil), t.storage)
}

func (t *testStorage) saveNewBlock(height base.Height) block.Block {
	blk, err := block.NewTestBlockV0(height, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)

	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	return blk
}

func (t *testStorage) TestLastBlock() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.LastBlock()
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testStorage) TestLastManifest() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
}

func (t *testStorage) TestLoadBlockByHash() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.Block(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testStorage) TestLoadBlockByHeight() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.BlockByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testStorage) TestLoadBlocksByHeight() {
	var blocks []block.Block
	heights := []base.Height{base.Height(33), base.Height(34)}
	for _, h := range heights {
		blk := t.saveNewBlock(h)

		blocks = append(blocks, blk)
	}

	loaded, err := t.storage.BlocksByHeight(heights)
	t.NoError(err)

	t.Equal(len(heights), len(blocks))

	for i := range heights {
		t.CompareBlock(blocks[i], loaded[i])
	}
}

func (t *testStorage) TestLoadManifestByHash() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testStorage) TestLoadManifestByHeight() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testStorage) TestSeals() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	for _, sl := range seals {
		found, err := t.storage.HasSeal(sl.Hash())
		t.NoError(err)
		t.True(found)
	}

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(
			[]byte(seals[i].Hash().String()),
			[]byte(seals[j].Hash().String()),
		) < 0
	})

	var collected []seal.Seal
	t.NoError(t.storage.Seals(
		func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
			collected = append(collected, sl)

			return true, nil
		},
		true,
		true,
	))

	t.Equal(len(seals), len(collected))

	for i, sl := range collected {
		t.True(seals[i].Hash().Equal(sl.Hash()))
	}
}

func (t *testStorage) TestSealsByHash() {
	var seals []seal.Seal
	var hashes []valuehash.Hash
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)
		hashes = append(hashes, sl.Hash())

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	loaded := map[valuehash.Hash]seal.Seal{}
	t.NoError(t.storage.SealsByHash(hashes, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
		loaded[sl.Hash()] = sl

		return true, nil
	}, true))

	for _, h := range hashes {
		_, found := loaded[h]
		t.True(found)
	}
}

func (t *testStorage) TestSealsOnlyHash() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(
			[]byte(seals[i].Hash().String()),
			[]byte(seals[j].Hash().String()),
		) < 0
	})

	var collected []valuehash.Hash
	t.NoError(t.storage.Seals(
		func(h valuehash.Hash, sl seal.Seal) (bool, error) {
			t.Nil(sl)
			collected = append(collected, h)

			return true, nil
		},
		true,
		false,
	))

	t.Equal(len(seals), len(collected))

	for i, h := range collected {
		t.True(seals[i].Hash().Equal(h))
	}
}

func (t *testStorage) TestSealsLimit() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(
			[]byte(seals[i].Hash().String()),
			[]byte(seals[j].Hash().String()),
		) < 0
	})

	var collected []seal.Seal
	t.NoError(t.storage.Seals(
		func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
			if len(collected) == 3 {
				return false, nil
			}

			collected = append(collected, sl)

			return true, nil
		},
		true,
		true,
	))

	t.Equal(3, len(collected))

	for i, sl := range collected {
		t.True(seals[i].Hash().Equal(sl.Hash()))
	}
}

func (t *testStorage) newOperationSeal() operation.Seal {
	token := []byte("this-is-token")
	op, err := operation.NewKVOperation(t.PK, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewSeal(t.PK, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testStorage) TestStagedOperationSeals() {
	var seals []seal.Seal

	// 10 seal.Seal
	for i := 0; i < 10; i++ {
		sl := seal.NewDummySeal(t.PK)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	ops := map[valuehash.Hash]operation.Seal{}
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()

		seals = append(seals, sl)
		ops[sl.Hash()] = sl
	}
	t.NoError(t.storage.NewSeals(seals))

	var collected []seal.Seal
	t.NoError(t.storage.StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			collected = append(collected, sl)

			return true, nil
		},
		true,
	))

	t.Equal(len(ops), len(collected))

	for _, sl := range collected {
		t.IsType(operation.Seal{}, sl)

		_, found := ops[sl.Hash()]
		t.True(found)
	}
}

func (t *testStorage) TestUnStagedOperationSeals() {
	// 10 seal.Seal
	for i := 0; i < 10; i++ {
		sl := seal.NewDummySeal(t.PK)
		t.NoError(t.storage.NewSeals([]seal.Seal{sl}))
	}

	var ops []operation.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()
		t.NoError(t.storage.NewSeals([]seal.Seal{sl}))

		ops = append(ops, sl)
	}

	var unstaged []valuehash.Hash

	rs := rand.New(rand.NewSource(time.Now().Unix()))
	selected := map[valuehash.Hash]struct{}{}
	for i := 0; i < 5; i++ {
		var sl seal.Seal
		for {
			sl = ops[rs.Intn(len(ops))]
			if _, found := selected[sl.Hash()]; !found {
				selected[sl.Hash()] = struct{}{}
				break
			}
		}
		unstaged = append(unstaged, sl.Hash())
	}

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)

	// unstage
	t.NoError(bs.UnstageOperationSeals(unstaged))
	t.NoError(bs.Commit())

	var collected []seal.Seal
	t.NoError(t.storage.StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			collected = append(collected, sl)

			return true, nil
		},
		true,
	))

	t.Equal(len(ops)-len(unstaged), len(collected))

	for _, sl := range collected {
		var found bool
		for _, usl := range unstaged {
			if sl.Hash().Equal(usl) {
				found = true
				break
			}
		}

		t.False(found)
	}
}

func (t *testStorage) TestHasOperation() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.storage.setLastBlock(blk, true)

	op, err := operation.NewKVOperation(t.PK, []byte("showme"), "key", []byte("value"), nil)
	t.NoError(err)

	{ // store
		doc, err := NewOperationDoc(op, t.storage.enc, base.Height(33))
		t.NoError(err)
		_, err = t.storage.client.Set("operation", doc)
		t.NoError(err)
	}

	{
		found, err := t.storage.HasOperation(op.Hash())
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.storage.HasOperation(valuehash.RandomSHA256())
		t.NoError(err)
		t.False(found)
	}
}

func (t *testStorage) TestCreateIndexNew() {
	allIndexes := func(col string) []string {
		iv := t.storage.client.Collection(col).Indexes()

		cursor, err := iv.List(context.TODO())
		t.NoError(err)

		var results []bson.M
		t.NoError(cursor.All(context.TODO(), &results))

		var names []string
		for _, r := range results {
			names = append(names, r["name"].(string))
		}

		return names
	}

	oldIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "showme", Value: 1}},
			Options: options.Index().SetName("mitum_showme"),
		},
	}

	t.NoError(t.storage.createIndex(defaultColNameManifest, oldIndexes))

	existings := allIndexes(defaultColNameManifest)

	newIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "findme", Value: 1}},
			Options: options.Index().SetName("mitum_findme"),
		},
	}

	t.NoError(t.storage.createIndex(defaultColNameManifest, newIndexes))
	created := allIndexes(defaultColNameManifest)

	t.Equal(existings, []string{"_id_", "mitum_showme"})
	t.Equal(created, []string{"_id_", "mitum_findme"})
}

func TestStorage(t *testing.T) {
	suite.Run(t, new(testStorage))
}
