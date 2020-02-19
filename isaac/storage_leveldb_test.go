package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/valuehash"
)

type testLeveldbStorage struct {
	suite.Suite
	localNode *LocalNode
	db        *leveldb.DB
	encs      *encoder.Encoders
	enc       encoder.Encoder
}

func (t *testLeveldbStorage) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(BlockType, "block")
	_ = hint.RegisterType(VoteproofType, "voteproof")

	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)
	_ = t.encs.AddHinter(BlockV0{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(VoteproofV0{})
}

func (t *testLeveldbStorage) SetupTest() {
	db, err := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	t.NoError(err)
	t.db = db
}

func (t *testLeveldbStorage) TestNew() {
	st := NewLeveldbStorage(t.db, t.encs, t.enc)
	t.NotNil(st)
	t.Implements((*Storage)(nil), st)
}

func (t *testLeveldbStorage) TestLoadBlock() {
	// store first
	block, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	st := NewLeveldbStorage(t.db, t.encs, t.enc)
	t.NotNil(st)

	{
		b, err := t.enc.Marshal(block)
		t.NoError(err)

		hb := storage.LeveldbDataWithEncoder(t.enc, b)

		t.NoError(t.db.Put(leveldbBlockKey(block), hb, nil))
	}

	loaded, err := st.LastBlock()
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
	st := NewLeveldbStorage(t.db, t.encs, t.enc)
	t.NotNil(st)

	{
		loaded, err := st.LastINITVoteproof()
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

	t.NoError(st.NewINITVoteproof(voteproof))

	loaded, err := st.LastINITVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(voteproof.Stage(), StageINIT)
	t.Equal(voteproof.Height(), loaded.Height())
	t.Equal(voteproof.Round(), loaded.Round())
	t.Equal(voteproof.Result(), loaded.Result())
	t.Equal(localtime.RFC3339(voteproof.FinishedAt()), localtime.RFC3339(loaded.FinishedAt()))
}

func (t *testLeveldbStorage) TestLoadACCEPTTVoteproof() {
	st := NewLeveldbStorage(t.db, t.encs, t.enc)
	t.NotNil(st)

	{
		loaded, err := st.LastINITVoteproof()
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

	t.NoError(st.NewINITVoteproof(ivp))

	avp := VoteproofV0{
		height:     Height(33),
		round:      Round(3),
		result:     VoteproofMajority,
		stage:      StageACCEPT,
		finishedAt: localtime.Now(),
	}
	t.NoError(st.NewACCEPTVoteproof(avp))

	loaded, err := st.LastACCEPTVoteproof()
	t.NoError(err)
	t.NotNil(loaded)

	t.Equal(avp.Stage(), StageACCEPT)
	t.Equal(avp.Height(), loaded.Height())
	t.Equal(avp.Round(), loaded.Round())
	t.Equal(avp.Result(), loaded.Result())
	t.Equal(localtime.RFC3339(avp.FinishedAt()), localtime.RFC3339(loaded.FinishedAt()))
}

func TestLeveldbStorage(t *testing.T) {
	suite.Run(t, new(testLeveldbStorage))
}
