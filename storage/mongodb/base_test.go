// +build mongodb

package mongodbstorage

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type testDatabase struct {
	storage.BaseTestDatabase
	database *Database
}

func (t *testDatabase) SetupTest() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	st, err := NewDatabase(client, t.Encs, nil, cache.Dummy{})
	t.NoError(err)
	t.database = st
}

func (t *testDatabase) TearDownTest() {
	if t.database != nil {
		t.database.Client().DropDatabase()
		t.database.Close()
	}
}

func (t *testDatabase) TestNew() {
	t.Implements((*storage.Database)(nil), t.database)
}

func (t *testDatabase) saveNewBlock(height base.Height) (block.Block, block.BlockDataMap) {
	blk, err := block.NewTestBlockV0(height, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	i := (interface{})(blk).(block.BlockUpdater)
	i = i.SetINITVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageINIT))
	i = i.SetACCEPTVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageACCEPT))
	blk = i.(block.BlockV0)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)

	t.NoError(bs.SetBlock(context.Background(), blk))
	bd := t.NewBlockDataMap(blk.Height(), blk.Hash(), true)
	t.NoError(bs.Commit(context.Background(), bd))

	return blk, bd
}

func (t *testDatabase) saveBlockDataMap(st *Database, bd block.BlockDataMap) error {
	if doc, err := NewBlockDataMapDoc(bd, st.enc); err != nil {
		return err
	} else if _, err := st.client.Add(ColNameBlockDataMap, doc); err != nil {
		return err
	} else {
		return nil
	}
}

func (t *testDatabase) TestLastBlock() {
	blk, bd := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.database.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)

	ubd, found, err := t.database.BlockDataMap(blk.Height())
	t.NoError(err)
	t.True(found)

	block.CompareBlockDataMap(t.Assert(), bd, ubd)
}

func (t *testDatabase) TestSetBlockContext() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*1)
	defer cancel()

	err = bs.SetBlock(ctx, blk)
	t.True(errors.Is(err, context.DeadlineExceeded))
}

func (t *testDatabase) TestSaveBlockContext() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	i := (interface{})(blk).(block.BlockUpdater)
	i = i.SetINITVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageINIT))
	i = i.SetACCEPTVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageACCEPT))
	blk = i.(block.BlockV0)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)

	t.NoError(bs.SetBlock(context.Background(), blk))

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*1)
	defer cancel()

	bd := t.NewBlockDataMap(blk.Height(), blk.Hash(), true)
	err = bs.Commit(ctx, bd)
	t.True(errors.Is(err, context.DeadlineExceeded))
}

func (t *testDatabase) TestLastManifest() {
	blk, _ := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.database.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
}

func (t *testDatabase) TestLoadManifestByHash() {
	blk, _ := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.database.Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestLoadManifestByHeight() {
	blk, _ := t.saveNewBlock(base.Height(33))

	loaded, found, err := t.database.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestSeals() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	for _, sl := range seals {
		found, err := t.database.HasSeal(sl.Hash())
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
	t.NoError(t.database.Seals(
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

func (t *testDatabase) TestSealsByHash() {
	var seals []seal.Seal
	var hashes []valuehash.Hash
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)
		hashes = append(hashes, sl.Hash())

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	loaded := map[string]seal.Seal{}
	t.NoError(t.database.SealsByHash(hashes, func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
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

func (t *testDatabase) TestSealsOnlyHash() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(
			[]byte(seals[i].Hash().String()),
			[]byte(seals[j].Hash().String()),
		) < 0
	})

	var collected []valuehash.Hash
	t.NoError(t.database.Seals(
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

func (t *testDatabase) TestSealsLimit() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(
			[]byte(seals[i].Hash().String()),
			[]byte(seals[j].Hash().String()),
		) < 0
	})

	var collected []seal.Seal
	t.NoError(t.database.Seals(
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

func (t *testDatabase) newOperationSeal() operation.Seal {
	token := []byte("this-is-token")
	op, err := operation.NewKVOperation(t.PK, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewBaseSeal(t.PK, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testDatabase) TestStagedOperationSeals() {
	var seals []seal.Seal

	// 10 seal.Seal
	for i := 0; i < 10; i++ {
		sl := seal.NewDummySeal(t.PK)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	ops := map[string]operation.Seal{}

	var others []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()

		others = append(others, sl)
		ops[sl.Hash().String()] = sl
	}
	t.NoError(t.database.NewSeals(others))

	var collected []seal.Seal
	t.NoError(t.database.StagedOperationSeals(
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

func (t *testDatabase) TestUnStagedOperationSeals() {
	// 10 seal.Seal
	for i := 0; i < 10; i++ {
		sl := seal.NewDummySeal(t.PK)
		t.NoError(t.database.NewSeals([]seal.Seal{sl}))
	}

	var ops []operation.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()
		t.NoError(t.database.NewSeals([]seal.Seal{sl}))

		ops = append(ops, sl)
	}

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
	}

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	i := (interface{})(blk).(block.BlockUpdater)
	i = i.SetINITVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageINIT))
	i = i.SetACCEPTVoteproof(base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageACCEPT))
	blk = i.(block.BlockV0)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)

	bd := t.NewBlockDataMap(blk.Height(), blk.Hash(), true)
	t.NoError(bs.Commit(context.Background(), bd))

	var collected []seal.Seal
	t.NoError(t.database.StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			collected = append(collected, sl)

			return true, nil
		},
		true,
	))

	t.Equal(len(ops), len(collected))
}

func (t *testDatabase) TestHasOperation() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.database.setLastBlock(blk, true, false)

	op, err := operation.NewKVOperation(t.PK, []byte("showme"), "key", []byte("value"), nil)
	t.NoError(err)

	{ // store
		doc, err := NewOperationDoc(op.Fact().Hash(), t.database.enc, base.Height(33))
		t.NoError(err)
		_, err = t.database.client.Set("operation", doc)
		t.NoError(err)
	}

	{
		found, err := t.database.HasOperationFact(op.Fact().Hash())
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.database.HasOperationFact(valuehash.RandomSHA256())
		t.NoError(err)
		t.False(found)
	}
}

func (t *testDatabase) TestCreateIndexNew() {
	allIndexes := func(col string) []string {
		iv := t.database.client.Collection(col).Indexes()

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

	t.NoError(t.database.CreateIndex(ColNameManifest, oldIndexes, IndexPrefix))

	existings := allIndexes(ColNameManifest)

	newIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{bson.E{Key: "findme", Value: 1}},
			Options: options.Index().SetName("mitum_findme"),
		},
	}

	t.NoError(t.database.CreateIndex(ColNameManifest, newIndexes, IndexPrefix))
	created := allIndexes(ColNameManifest)

	t.Equal(existings, []string{"_id_", "mitum_showme"})
	t.Equal(created, []string{"_id_", "mitum_findme"})
}

func (t *testDatabase) TestCopy() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)

	other, err := NewDatabase(client, t.Encs, t.BSONEnc, cache.Dummy{})
	t.NoError(err)
	defer func() {
		other.Client().DropDatabase()
		other.Close()
	}()

	var cols []string
	for i := 0; i < 3; i++ {
		col := util.UUID().String()
		doc := NewDocNilID(nil, bson.M{"v": i})
		_, err = t.database.Client().Set(col, doc)
		t.NoError(err)

		cols = append(cols, col)
	}

	sort.Strings(cols)

	t.NoError(other.Copy(t.database))

	otherCols, err := other.Client().Collections()
	t.NoError(err)

	sort.Strings(otherCols)

	t.Equal(cols, otherCols)

	for _, col := range cols {
		rs := map[string]bson.M{}
		t.NoError(t.database.Client().Find(context.TODO(), col, bson.M{}, func(cursor *mongo.Cursor) (bool, error) {
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

func (t *testDatabase) TestInfo() {
	key := util.UUID().String()
	b := util.UUID().Bytes()

	_, found, err := t.database.Info(key)
	t.Nil(err)
	t.False(found)

	t.NoError(t.database.SetInfo(key, b))

	ub, found, err := t.database.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(b, ub)

	nb := util.UUID().Bytes()
	t.NoError(t.database.SetInfo(key, nb))

	unb, found, err := t.database.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(nb, unb)
}

func (t *testDatabase) TestLocalBlockDataMapsByHeight() {
	isLocal := func(height base.Height) bool {
		switch height {
		case 33, 36, 39:
			return true
		default:
			return false
		}
	}

	for i := base.Height(33); i < 40; i++ {
		bd := t.NewBlockDataMap(i, valuehash.RandomSHA256(), isLocal(i))
		t.NoError(t.saveBlockDataMap(t.database, bd))
	}

	for i := base.Height(33); i < 40; i++ {
		bd, found, err := t.database.BlockDataMap(i)
		t.NoError(err)
		t.True(found)
		t.NotNil(bd)
	}

	err := t.database.LocalBlockDataMapsByHeight(36, func(bd block.BlockDataMap) (bool, error) {
		t.True(bd.Height() > 35)
		t.True(isLocal(bd.Height()))
		t.True(bd.IsLocal())

		return true, nil
	})
	t.NoError(err)
}

func TestMongodbDatabase(t *testing.T) {
	suite.Run(t, new(testDatabase))
}
