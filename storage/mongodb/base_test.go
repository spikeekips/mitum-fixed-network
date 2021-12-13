//go:build mongodb
// +build mongodb

package mongodbstorage

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
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

func (t *testDatabase) TestNewOperationSeals() {
	var seals []operation.Seal
	var ops []operation.Operation
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()
		seals = append(seals, sl)

		k := sl.Operations()
		for j := range k {
			ops = append(ops, k[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	for i := range ops {
		found, err := t.database.HasStagedOperation(ops[i].Fact().Hash())
		t.NoError(err)
		t.True(found)
	}

	var collected []operation.Operation
	t.NoError(t.database.StagedOperations(
		func(op operation.Operation) (bool, error) {
			collected = append(collected, op)

			return true, nil
		},
		true,
	))

	t.Equal(len(ops), len(collected))

	for i := range collected {
		t.True(ops[i].Fact().Hash().Equal(collected[i].Fact().Hash()))
	}
}

func (t *testDatabase) TestStagedOperationsByFact() {
	var seals []operation.Seal
	var ops []operation.Operation
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()
		seals = append(seals, sl)

		k := sl.Operations()
		for j := range k {
			ops = append(ops, k[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	for i := range ops {
		found, err := t.database.HasStagedOperation(ops[i].Fact().Hash())
		t.NoError(err)
		t.True(found)
	}

	var facts []valuehash.Hash
	for i := range ops[:3] {
		facts = append(facts, ops[i].Fact().Hash())
	}

	rops, err := t.database.StagedOperationsByFact(facts)
	t.NoError(err)
	for i := range rops {
		op := rops[i]

		t.True(facts[i].Equal(op.Fact().Hash()))
	}

	rops, err = t.database.StagedOperationsByFact([]valuehash.Hash{valuehash.RandomSHA256()})
	t.NoError(err)
	t.Equal(0, len(rops))
}

func (t *testDatabase) TestStagedOperationsLimit() {
	var seals []operation.Seal
	var ops []operation.Operation
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()
		seals = append(seals, sl)

		k := sl.Operations()
		for j := range k {
			ops = append(ops, k[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	for i := range ops {
		found, err := t.database.HasStagedOperation(ops[i].Fact().Hash())
		t.NoError(err)
		t.True(found)
	}

	var collected []operation.Operation
	t.NoError(t.database.StagedOperations(
		func(op operation.Operation) (bool, error) {
			if len(collected) == 3 {
				return false, nil
			}

			collected = append(collected, op)

			return true, nil
		},
		true,
	))

	t.Equal(3, len(collected))

	for i := range collected {
		t.True(ops[i].Fact().Hash().Equal(collected[i].Fact().Hash()))
	}
}

func (t *testDatabase) TestUnstagedOperations() {
	var ops []operation.Operation
	var seals []operation.Seal
	for i := 0; i < 10; i++ {
		opsl := t.newOperationSeal()
		seals = append(seals, opsl)

		l := opsl.Operations()
		for j := range l {
			ops = append(ops, l[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	var inserted int
	_ = t.database.client.Find(
		context.Background(),
		ColNameStagedOperation,
		func(_ *mongo.Cursor) (bool, error) {
			inserted++
			return true, nil
		},
		nil,
	)
	t.Equal(0, inserted)

	var facts []valuehash.Hash
	for i := range ops[:3] {
		facts = append(facts, ops[i].Fact().Hash())
	}

	t.NoError(t.database.UnstagedOperations(facts))

	for i := range facts {
		found, err := t.database.HasStagedOperation(facts[i])
		t.NoError(err)
		t.False(found)
	}

	rops, err := t.database.StagedOperationsByFact(facts)
	t.NoError(err)
	t.Equal(0, len(rops))

	var lefts int
	_ = t.database.client.Find(
		context.Background(),
		ColNameStagedOperation,
		func(_ *mongo.Cursor) (bool, error) {
			lefts++
			return true, nil
		},
		nil,
	)

	t.Equal(inserted, lefts)
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
