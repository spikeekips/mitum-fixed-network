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
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type testLeveldbStorage struct {
	suite.Suite
	encs    *encoder.Encoders
	enc     encoder.Encoder
	storage *LeveldbStorage
	pk      key.BTCPrivatekey
}

func (t *testLeveldbStorage) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(block.BlockV0{})
	_ = t.encs.AddHinter(block.ManifestV0{})
	_ = t.encs.AddHinter(block.BlockConsensusInfoV0{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(base.VoteproofV0{})
	_ = t.encs.AddHinter(seal.DummySeal{})
	_ = t.encs.AddHinter(operation.Seal{})
	_ = t.encs.AddHinter(operation.KVOperation{})
	_ = t.encs.AddHinter(operation.KVOperationFact{})

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testLeveldbStorage) SetupTest() {
	t.storage = NewMemStorage(t.encs, t.enc)
}

func (t *testLeveldbStorage) compareManifest(a, b block.Manifest) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Proposal().Equal(b.Proposal()))
	t.True(a.PreviousBlock().Equal(b.PreviousBlock()))
	t.True(a.OperationsHash().Equal(b.OperationsHash()))
	t.True(a.StatesHash().Equal(b.StatesHash()))
}

func (t *testLeveldbStorage) compareBlock(a, b block.Block) {
	t.compareManifest(a, b)
	t.Equal(a.INITVoteproof(), b.INITVoteproof())
	t.Equal(a.ACCEPTVoteproof(), b.ACCEPTVoteproof())
}

func (t *testLeveldbStorage) TestNew() {
	t.Implements((*storage.Storage)(nil), t.storage)
}

func (t *testLeveldbStorage) TestLastBlock() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.LastBlock()
	t.NoError(err)

	t.compareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadBlockByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	{
		b, err := t.enc.Marshal(blk)
		t.NoError(err)

		hb := storage.LeveldbDataWithEncoder(t.enc, b)

		key := leveldbBlockHashKey(blk.Hash())
		t.NoError(t.storage.db.Put(key, hb, nil))
		t.NoError(t.storage.db.Put(leveldbBlockHeightKey(blk.Height()), key, nil))
	}

	loaded, err := t.storage.Block(blk.Hash())
	t.NoError(err)

	t.compareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadManifestByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.Manifest(blk.Hash())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.compareManifest(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadManifestByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.compareManifest(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadBlockByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	bs, err := t.storage.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(blk))
	t.NoError(bs.Commit())

	loaded, err := t.storage.BlockByHeight(blk.Height())
	t.NoError(err)

	t.compareBlock(blk, loaded)
}

func (t *testLeveldbStorage) TestLoadINITVoteproof() {
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
	t.Equal(localtime.RFC3339(voteproof.FinishedAt()), localtime.RFC3339(loaded.FinishedAt()))
}

func (t *testLeveldbStorage) TestLoadACCEPTVoteproof() {
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
	t.Equal(localtime.RFC3339(avp.FinishedAt()), localtime.RFC3339(loaded.FinishedAt()))
}

func (t *testLeveldbStorage) TestLoadVoteproofs() {
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
	op, err := operation.NewKVOperation(t.pk, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewSeal(t.pk, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testLeveldbStorage) TestStagedOperationSeals() {
	var seals []seal.Seal

	// 10 seal.Seal
	for i := 0; i < 10; i++ {
		sl := seal.NewDummySeal(t.pk)

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
		sl := seal.NewDummySeal(t.pk)
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
		raw, err := t.storage.enc.Encode(op)
		t.NoError(err)
		t.storage.db.Put(
			leveldbOperationHashKey(op),
			storage.LeveldbDataWithEncoder(t.storage.enc, raw),
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
