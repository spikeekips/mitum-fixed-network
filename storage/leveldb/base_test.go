package leveldbstorage

import (
	"bytes"
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testDatabase struct {
	storage.BaseTestDatabase
	database *Database
}

func (t *testDatabase) SetupTest() {
	t.database = NewMemDatabase(t.Encs, t.JSONEnc)
}

func (t *testDatabase) TestNew() {
	t.Implements((*storage.Database)(nil), t.database)
}

func (t *testDatabase) saveBlockDataMap(st *Database, bd block.BlockDataMap) error {
	if b, err := marshal(st.enc, bd); err != nil {
		return err
	} else {
		return st.db.Put(leveldbBlockDataMapKey(bd.Height()), b, nil)
	}
}

func (t *testDatabase) TestLastBlock() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))

	bd := t.NewBlockDataMap(blk.Height(), blk.Hash(), true)
	t.NoError(bs.Commit(context.Background(), bd))

	loaded, found, err := t.database.lastBlock()
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)

	ubd, found, err := t.database.BlockDataMap(blk.Height())
	t.NoError(err)
	t.True(found)

	block.CompareBlockDataMap(t.Assert(), bd, ubd)
}

func (t *testDatabase) TestLastManifest() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockDataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
}

func (t *testDatabase) TestLoadBlockByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	{
		b, err := t.JSONEnc.Marshal(blk)
		t.NoError(err)

		hb := encodeWithEncoder(t.JSONEnc, b)

		key := leveldbBlockHashKey(blk.Hash())
		t.NoError(t.database.db.Put(key, hb, nil))
		t.NoError(t.database.db.Put(leveldbBlockHeightKey(blk.Height()), key, nil))
	}

	loaded, found, err := t.database.block(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testDatabase) TestLoadManifestByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockDataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestLoadManifestByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockDataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestLoadBlockByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockDataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.blockByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testDatabase) TestSeals() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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

func (t *testDatabase) TestSealsOnlyHash() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.database.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal()

		seals = append(seals, sl)
		ops[sl.Hash().String()] = sl
	}
	t.NoError(t.database.NewSeals(seals))

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

	bs, err := t.database.NewSession(blk)
	t.NoError(err)

	// unstage
	t.NoError(bs.Commit(context.Background(), t.NewBlockDataMap(blk.Height(), blk.Hash(), true)))

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
	fact := valuehash.RandomSHA256()

	{ // store
		raw, err := t.database.enc.Marshal(fact)
		t.NoError(err)
		t.database.db.Put(
			leveldbOperationFactHashKey(fact),
			encodeWithEncoder(t.database.enc, raw),
			nil,
		)
	}

	{
		found, err := t.database.HasOperationFact(fact)
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.database.HasOperationFact(valuehash.RandomSHA256())
		t.NoError(err)
		t.False(found)
	}
}

func (t *testDatabase) TestInfo() {
	key := util.UUID().String()
	b := util.UUID().Bytes()

	_, found, err := t.database.Info(key)
	t.NoError(err)
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

func TestLeveldbDatabase(t *testing.T) {
	suite.Run(t, new(testDatabase))
}
