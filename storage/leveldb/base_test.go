package leveldbstorage

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testLeveldbStorage struct {
	storage.BaseTestStorage
	storage *Storage
}

func (t *testLeveldbStorage) SetupTest() {
	t.storage = NewMemStorage(t.Encs, t.JSONEnc)
}

func (t *testLeveldbStorage) TestNew() {
	t.Implements((*storage.Storage)(nil), t.storage)
}

func (t *testLeveldbStorage) TestLastBlock() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.LastBlock()
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadBlockByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	{
		b, err := t.JSONEnc.Marshal(blk)
		t.NoError(err)

		hb := encodeWithEncoder(t.JSONEnc, b)

		key := leveldbBlockHashKey(blk.Hash())
		t.NoError(t.storage.db.Put(key, hb, nil))
		t.NoError(t.storage.db.Put(leveldbBlockHeightKey(blk.Height()), key, nil))
	}

	loaded, err := t.storage.Block(blk.Hash())
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadManifestByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.Manifest(blk.Hash())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadManifestByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadBlockByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.BlockByHeight(blk.Height())
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestSeals() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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

func (t *testLeveldbStorage) TestSealsOnlyHash() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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

func (t *testLeveldbStorage) TestSealsLimit() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)

		seals = append(seals, sl)
	}
	t.NoError(t.storage.NewSeals(seals))

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
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

func (t *testLeveldbStorage) newOperationSeal() operation.Seal {
	token := []byte("this-is-token")
	op, err := operation.NewKVOperation(t.PK, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewSeal(t.PK, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testLeveldbStorage) TestStagedOperationSeals() {
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

func (t *testLeveldbStorage) TestUnStagedOperationSeals() {
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

	// reverse key also will be removed
	for _, h := range unstaged {
		_, err := t.storage.get(t.storage.newStagedOperationSealReverseKey(h))
		t.True(xerrors.Is(err, storage.NotFoundError))
	}
}

func (t *testLeveldbStorage) TestHasOperation() {
	op := valuehash.RandomSHA256()

	{ // store
		raw, err := t.storage.enc.Marshal(op)
		t.NoError(err)
		t.storage.db.Put(
			leveldbOperationHashKey(op),
			encodeWithEncoder(t.storage.enc, raw),
			nil,
		)
	}

	{
		found, err := t.storage.HasOperation(op)
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.storage.HasOperation(valuehash.RandomSHA256())
		t.NoError(err)
		t.False(found)
	}
}

func TestLeveldbStorage(t *testing.T) {
	suite.Run(t, new(testLeveldbStorage))
}
