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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testStorage struct {
	storage.BaseTestStorage
	storage *Storage
}

func (t *testStorage) SetupTest() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	st, err := NewStorage(client, t.Encs, nil)
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
	t.NoError(bs.Commit(context.Background()))

	return blk
}

func (t *testStorage) TestLastBlock() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
}

func (t *testStorage) TestSaveBlockContext() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)

	t.NoError(bs.SetBlock(blk))

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*1)
	defer cancel()
	err = bs.Commit(ctx)
	t.True(xerrors.Is(err, context.DeadlineExceeded))
}

func (t *testStorage) TestLastManifest() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.storage.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
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

	loaded := map[string]seal.Seal{}
	t.NoError(t.storage.SealsByHash(hashes, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
		loaded[sl.Hash().String()] = sl

		return true, nil
	}, true))

	for _, h := range hashes {
		var found bool
		for lh := range loaded {
			if h.String() == lh {
				found = true
				break
			}
		}
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

	sl, err := operation.NewBaseSeal(t.PK, []operation.Operation{op}, nil)
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

	ops := map[string]operation.Seal{}

	var others []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()

		others = append(others, sl)
		ops[sl.Hash().String()] = sl
	}
	t.NoError(t.storage.NewSeals(others))

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
		t.Implements((*operation.Seal)(nil), sl)

		var found bool
		for h := range ops {
			if sl.Hash().String() == h {
				found = true
				break
			}
		}

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
	selected := map[string]struct{}{}
	for i := 0; i < 5; i++ {
		var sl seal.Seal
		for {
			sl = ops[rs.Intn(len(ops))]
			if _, found := selected[sl.Hash().String()]; !found {
				selected[sl.Hash().String()] = struct{}{}
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
	t.NoError(bs.Commit(context.Background()))

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

	t.storage.setLastBlock(blk, true, false)

	op, err := operation.NewKVOperation(t.PK, []byte("showme"), "key", []byte("value"), nil)
	t.NoError(err)

	{ // store
		doc, err := NewOperationDoc(op.Fact().Hash(), t.storage.enc, base.Height(33))
		t.NoError(err)
		_, err = t.storage.client.Set("operation", doc)
		t.NoError(err)
	}

	{
		found, err := t.storage.HasOperationFact(op.Fact().Hash())
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.storage.HasOperationFact(valuehash.RandomSHA256())
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

func (t *testStorage) TestCopy() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	other, err := NewStorage(client, t.Encs, t.BSONEnc)
	t.NoError(err)

	var cols []string
	for i := 0; i < 3; i++ {
		col := util.UUID().String()
		doc := NewDocNilID(nil, bson.M{"v": i})
		_, err = t.storage.Client().Set(col, doc)
		t.NoError(err)

		cols = append(cols, col)
	}

	sort.Strings(cols)

	t.NoError(other.Copy(t.storage))

	otherCols, err := other.Client().Collections()
	t.NoError(err)

	sort.Strings(otherCols)

	t.Equal(cols, otherCols)

	for _, col := range cols {
		rs := map[string]bson.M{}
		t.NoError(t.storage.Client().Find(nil, col, bson.M{}, func(cursor *mongo.Cursor) (bool, error) {
			var record bson.M
			if err := cursor.Decode(&record); err != nil {
				return false, err
			} else {
				rs[record["_id"].(primitive.ObjectID).Hex()] = record
			}

			return true, nil
		}))

		for sid, record := range rs {
			oid, err := primitive.ObjectIDFromHex(sid)
			t.NoError(err)

			var doc bson.M
			t.NoError(other.Client().GetByID(col, oid, func(res *mongo.SingleResult) error {
				return res.Decode(&doc)
			}))

			t.Equal(record["v"], doc["v"])
		}
	}
}

func (t *testStorage) TestInfo() {
	key := util.UUID().String()
	b := util.UUID().Bytes()

	_, found, err := t.storage.Info(key)
	t.Nil(err)
	t.False(found)

	t.NoError(t.storage.SetInfo(key, b))

	ub, found, err := t.storage.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(b, ub)

	nb := util.UUID().Bytes()
	t.NoError(t.storage.SetInfo(key, nb))

	unb, found, err := t.storage.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(nb, unb)
}

func TestMongodbStorage(t *testing.T) {
	suite.Run(t, new(testStorage))
}
