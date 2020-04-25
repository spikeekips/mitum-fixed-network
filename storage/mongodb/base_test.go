// +build mongodb

package mongodbstorage

import (
	"bytes"
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
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
)

type testMongodbStorage struct {
	storage.BaseTestStorage
	storage *MongodbStorage
}

func (t *testMongodbStorage) SetupTest() {
	client, err := NewClient(TestMongodbURI(), time.Second*2, time.Second*2)
	t.NoError(err)
	t.storage = NewMongodbStorage(client, t.Encs, t.BSONEnc)
}

func (t *testMongodbStorage) TestNew() {
	t.Implements((*storage.Storage)(nil), t.storage)
}

func (t *testMongodbStorage) saveNewBlock(height base.Height) block.Block {
	blk, err := block.NewTestBlockV0(height, base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)

	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	return blk
}

func (t *testMongodbStorage) TestLastBlock() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, err := t.storage.LastBlock()
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testMongodbStorage) TestLoadBlockByHash() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, err := t.storage.Block(blk.Hash())
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testMongodbStorage) TestLoadBlockByHeight() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, err := t.storage.BlockByHeight(blk.Height())
	t.NoError(err)

	t.CompareBlock(blk, loaded)
}

func (t *testMongodbStorage) TestLoadManifestByHash() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, err := t.storage.Manifest(blk.Hash())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testMongodbStorage) TestLoadManifestByHeight() {
	blk := t.saveNewBlock(base.Height(33))

	loaded, err := t.storage.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testMongodbStorage) TestLoadINITVoteproof() {
	{
		loaded, err := t.storage.LastINITVoteproof()
		t.Nil(err)
		t.Nil(loaded)
	}

	// store first
	threshold, _ := base.NewThreshold(2, 67)
	voteproof := base.NewVoteproofV0(
		base.Height(33),
		base.Round(3),
		threshold,
		base.StageINIT,
	)
	voteproof.SetResult(base.VoteResultMajority).Finish()

	t.NoError(t.storage.NewINITVoteproof(voteproof))

	loaded, err := t.storage.LastINITVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(voteproof.Stage(), base.StageINIT)
	t.Equal(voteproof.Height(), loaded.Height())
	t.Equal(voteproof.Round(), loaded.Round())
	t.Equal(voteproof.Result(), loaded.Result())
	t.Equal(localtime.Normalize(voteproof.FinishedAt()), localtime.Normalize(loaded.FinishedAt()))
}

func (t *testMongodbStorage) TestLoadACCEPTVoteproof() {
	{
		loaded, err := t.storage.LastINITVoteproof()
		t.Nil(err)
		t.Nil(loaded)
	}

	// store first
	threshold, _ := base.NewThreshold(2, 67)
	ivp := base.NewVoteproofV0(
		base.Height(33),
		base.Round(3),
		threshold,
		base.StageINIT,
	)

	ivp.SetResult(base.VoteResultMajority).Finish()

	t.NoError(t.storage.NewINITVoteproof(ivp))

	avp := base.NewVoteproofV0(
		base.Height(33),
		base.Round(3),
		threshold,
		base.StageACCEPT,
	)
	avp.SetResult(base.VoteResultMajority).Finish()

	t.NoError(t.storage.NewACCEPTVoteproof(avp))

	loaded, err := t.storage.LastACCEPTVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(avp.Stage(), base.StageACCEPT)
	t.Equal(avp.Height(), loaded.Height())
	t.Equal(avp.Round(), loaded.Round())
	t.Equal(avp.Result(), loaded.Result())
	t.Equal(localtime.Normalize(avp.FinishedAt()), localtime.Normalize(loaded.FinishedAt()))
}

func (t *testMongodbStorage) TestLoadVoteproofs() {
	{
		loaded, err := t.storage.LastINITVoteproof()
		t.Nil(err)
		t.Nil(loaded)
	}

	// store first
	threshold, _ := base.NewThreshold(2, 67)
	ivp := base.NewVoteproofV0(
		base.Height(33),
		base.Round(3),
		threshold,
		base.StageINIT,
	)
	ivp.SetResult(base.VoteResultMajority).Finish()

	t.NoError(t.storage.NewINITVoteproof(ivp))

	avp := base.NewVoteproofV0(
		base.Height(33),
		base.Round(3),
		threshold,
		base.StageACCEPT,
	)
	avp.SetResult(base.VoteResultMajority).Finish()

	t.NoError(t.storage.NewACCEPTVoteproof(avp))

	loaded, err := t.storage.LastACCEPTVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	var voteproofs []base.Voteproof
	t.storage.Voteproofs(func(voteproof base.Voteproof) (bool, error) {
		voteproofs = append(voteproofs, voteproof)

		return true, nil
	}, false)

	t.Equal(2, len(voteproofs))
}

func (t *testMongodbStorage) TestSeals() {
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

func (t *testMongodbStorage) TestSealsOnlyHash() {
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

func (t *testMongodbStorage) TestSealsLimit() {
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

func (t *testMongodbStorage) newOperationSeal() operation.Seal {
	token := []byte("this-is-token")
	op, err := operation.NewKVOperation(t.PK, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewSeal(t.PK, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testMongodbStorage) TestStagedOperationSeals() {
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

func (t *testMongodbStorage) TestUnStagedOperationSeals() {
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

func (t *testMongodbStorage) TestHasOperation() {
	op, err := operation.NewKVOperation(t.PK, []byte("showme"), "key", []byte("value"), nil)
	t.NoError(err)

	{ // store
		doc, err := NewOperationDoc(op, t.storage.enc)
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

func TestMongodbStorage(t *testing.T) {
	suite.Run(t, new(testMongodbStorage))
}
