package isaac

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type testLeveldbStorage struct {
	suite.Suite
	localNode *LocalNode
	encs      *encoder.Encoders
	enc       encoder.Encoder
	storage   *LeveldbStorage
	pk        key.BTCPrivatekey
}

func (t *testLeveldbStorage) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(BlockV0{})
	_ = t.encs.AddHinter(BlockManifestV0{})
	_ = t.encs.AddHinter(BlockConsensusInfoV0{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(VoteproofV0{})
	_ = t.encs.AddHinter(seal.DummySeal{})
	_ = t.encs.AddHinter(operation.Seal{})
	_ = t.encs.AddHinter(operation.KVOperation{})
	_ = t.encs.AddHinter(operation.KVOperationFact{})

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testLeveldbStorage) SetupTest() {
	t.storage = NewMemStorage(t.encs, t.enc)
}

func (t *testLeveldbStorage) TestNew() {
	t.Implements((*Storage)(nil), t.storage)
}

func (t *testLeveldbStorage) TestLastBlock() {
	// store first
	block, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	{
		b, err := t.enc.Marshal(block)
		t.NoError(err)

		hb := storage.LeveldbDataWithEncoder(t.enc, b)

		key := leveldbBlockHashKey(block.Hash())
		t.NoError(t.storage.db.Put(key, hb, nil))
		t.NoError(t.storage.db.Put(leveldbBlockHeightKey(block.Height()), key, nil))
	}

	loaded, err := t.storage.LastBlock()
	t.NoError(err)

	t.Equal(block.Height(), loaded.Height())
	t.Equal(block.Round(), loaded.Round())
	t.True(block.Proposal().Equal(loaded.Proposal()))
	t.True(block.PreviousBlock().Equal(loaded.PreviousBlock()))
	t.True(block.Operations().Equal(loaded.Operations()))
	t.True(block.States().Equal(loaded.States()))
	t.Equal(block.INITVoteproof(), loaded.INITVoteproof())
	t.Equal(block.ACCEPTVoteproof(), loaded.ACCEPTVoteproof())
}

func (t *testLeveldbStorage) TestLoadBlockByHash() {
	// store first
	block, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	{
		b, err := t.enc.Marshal(block)
		t.NoError(err)

		hb := storage.LeveldbDataWithEncoder(t.enc, b)

		key := leveldbBlockHashKey(block.Hash())
		t.NoError(t.storage.db.Put(key, hb, nil))
		t.NoError(t.storage.db.Put(leveldbBlockHeightKey(block.Height()), key, nil))
	}

	loaded, err := t.storage.Block(block.Hash())
	t.NoError(err)

	t.Equal(block.Height(), loaded.Height())
	t.Equal(block.Round(), loaded.Round())
	t.True(block.Proposal().Equal(loaded.Proposal()))
	t.True(block.PreviousBlock().Equal(loaded.PreviousBlock()))
	t.True(block.Operations().Equal(loaded.Operations()))
	t.True(block.States().Equal(loaded.States()))
	t.Equal(block.INITVoteproof(), loaded.INITVoteproof())
	t.Equal(block.ACCEPTVoteproof(), loaded.ACCEPTVoteproof())
}

func (t *testLeveldbStorage) TestLoadBlockByHeight() {
	// store first
	block, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	t.NotNil(t.storage)

	{
		b, err := t.enc.Marshal(block)
		t.NoError(err)

		hb := storage.LeveldbDataWithEncoder(t.enc, b)

		key := leveldbBlockHashKey(block.Hash())
		t.NoError(t.storage.db.Put(key, hb, nil))
		t.NoError(t.storage.db.Put(leveldbBlockHeightKey(block.Height()), key, nil))
	}

	loaded, err := t.storage.BlockByHeight(block.Height())
	t.NoError(err)

	t.Equal(block.Height(), loaded.Height())
	t.Equal(block.Round(), loaded.Round())
	t.True(block.Proposal().Equal(loaded.Proposal()))
	t.True(block.PreviousBlock().Equal(loaded.PreviousBlock()))
	t.True(block.Operations().Equal(loaded.Operations()))
	t.True(block.States().Equal(loaded.States()))
	t.Equal(block.INITVoteproof(), loaded.INITVoteproof())
	t.Equal(block.ACCEPTVoteproof(), loaded.ACCEPTVoteproof())
}

func (t *testLeveldbStorage) TestLoadINITVoteproof() {
	{
		loaded, err := t.storage.LastINITVoteproof()
		t.Nil(err)
		t.Nil(loaded)
	}

	// store first
	voteproof := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageINIT,
		finishedAt: localtime.Now(),
	}

	t.NoError(t.storage.NewINITVoteproof(voteproof))

	loaded, err := t.storage.LastINITVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(voteproof.Stage(), StageINIT)
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
	ivp := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageINIT,
		finishedAt: localtime.Now(),
	}

	t.NoError(t.storage.NewINITVoteproof(ivp))

	avp := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageACCEPT,
		finishedAt: localtime.Now(),
	}
	t.NoError(t.storage.NewACCEPTVoteproof(avp))

	loaded, err := t.storage.LastACCEPTVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(avp.Stage(), StageACCEPT)
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
	ivp := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageINIT,
		finishedAt: localtime.Now(),
	}

	t.NoError(t.storage.NewINITVoteproof(ivp))

	avp := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageACCEPT,
		finishedAt: localtime.Now(),
	}
	t.NoError(t.storage.NewACCEPTVoteproof(avp))

	loaded, err := t.storage.LastACCEPTVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	var voteproofs []Voteproof
	t.storage.Voteproofs(func(voteproof Voteproof) (bool, error) {
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

	block, err := NewTestBlockV0(Height(33), Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.storage.OpenBlockStorage(block)
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
