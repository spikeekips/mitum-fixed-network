package isaac

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/valuehash"
)

type testLeveldbStorage struct {
	suite.Suite
	localNode *LocalNode
	encs      *encoder.Encoders
	enc       encoder.Encoder
	storage   *LeveldbStorage
}

func (t *testLeveldbStorage) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(BlockType, "block")
	_ = hint.RegisterType(VoteproofType, "voteproof")
	_ = hint.RegisterType(seal.DummySeal{}.Hint().Type(), "dummy-seal")

	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)
	_ = t.encs.AddHinter(BlockV0{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(VoteproofV0{})
	_ = t.encs.AddHinter(seal.DummySeal{})
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
		t.NoError(t.storage.NewSeal(sl))

		seals = append(seals, sl)
	}

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
	})

	var collected []seal.Seal
	t.NoError(t.storage.Seals(
		func(sl seal.Seal) (bool, error) {
			collected = append(collected, sl)

			return true, nil
		},
		true,
	))

	t.Equal(len(seals), len(collected))

	for i, sl := range collected {
		t.True(seals[i].Hash().Equal(sl.Hash()))
	}
}

func (t *testLeveldbStorage) TestSealsLimit() {
	var seals []seal.Seal
	for i := 0; i < 10; i++ {
		pk, _ := key.NewBTCPrivatekey()
		sl := seal.NewDummySeal(pk)
		t.NoError(t.storage.NewSeal(sl))

		seals = append(seals, sl)
	}

	sort.Slice(seals, func(i, j int) bool {
		return bytes.Compare(seals[i].Hash().Bytes(), seals[j].Hash().Bytes()) < 0
	})

	var collected []seal.Seal
	t.NoError(t.storage.Seals(
		func(sl seal.Seal) (bool, error) {
			if len(collected) == 3 {
				return false, nil
			}

			collected = append(collected, sl)

			return true, nil
		},
		true,
	))

	t.Equal(3, len(collected))

	for i, sl := range collected {
		t.True(seals[i].Hash().Equal(sl.Hash()))
	}
}

func TestLeveldbStorage(t *testing.T) {
	suite.Run(t, new(testLeveldbStorage))
}
