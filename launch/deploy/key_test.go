package deploy

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testDeployKeyStorage struct {
	suite.Suite
}

func (t *testDeployKeyStorage) TestNew() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)
	t.Equal(0, ks.Len())
}

func (t *testDeployKeyStorage) TestAdd() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)
	t.Equal(0, ks.Len())

	nk, err := ks.New()
	t.NoError(err)
	t.Equal(1, ks.Len())
	t.True(ks.Exists(nk.Key()))

	unk, found := ks.Key(nk.Key())
	t.True(found)
	t.Equal(nk.Key(), unk.Key())

	unknown, found := ks.Key(util.UUID().String())
	t.False(found)
	t.Empty(unknown.Key())

	var added []DeployKey
	ks.Traverse(func(k DeployKey) bool {
		added = append(added, k)

		return true
	})

	t.Equal(1, len(added))
}

func (t *testDeployKeyStorage) TestRevoke() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)
	t.Equal(0, ks.Len())

	nk, err := ks.New()
	t.NoError(err)

	t.NoError(err)
	t.Equal(1, ks.Len())

	t.NoError(ks.Revoke(nk.Key()))

	t.Equal(0, ks.Len())
}

func (t *testDeployKeyStorage) TestRevokeUnknown() {
	ks, err := NewDeployKeyStorage(nil)
	t.NoError(err)
	t.Equal(0, ks.Len())

	_, err = ks.New()
	t.NoError(err)

	t.Equal(1, ks.Len())

	err = ks.Revoke(util.UUID().String())
	t.True(xerrors.Is(err, util.NotFoundError))

	t.Equal(1, ks.Len())
}

func TestDeployKeyStorage(t *testing.T) {
	suite.Run(t, new(testDeployKeyStorage))
}

type testDeployKeyStorageWithDatabase struct {
	suite.Suite
	isaac.StorageSupportTest
	db storage.Database
}

func (t *testDeployKeyStorageWithDatabase) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.NoError(t.Encs.AddHinter(key.BTCPublickeyHinter))
}

func (t *testDeployKeyStorageWithDatabase) SetupTest() {
	t.db = t.Database(t.Encs, nil)
}

func (t *testDeployKeyStorageWithDatabase) TestAdd() {
	ks, err := NewDeployKeyStorage(t.db)
	t.NoError(err)
	t.Equal(0, ks.Len())

	for i := 0; i < 3; i++ {
		_, err = ks.New()
		t.NoError(err)
	}

	// NOTE check in info
	uks, err := loadDeployKeys(t.db)
	t.NoError(err)
	for i := range ks.keys {
		a := ks.keys[i]
		b := uks[i]

		t.Equal(a.Key(), b.Key())
		t.True(localtime.Equal(a.AddedAt(), b.AddedAt()))
	}
}

func (t *testDeployKeyStorageWithDatabase) TestLoad() {
	ks, err := NewDeployKeyStorage(t.db)
	t.NoError(err)
	t.Equal(0, ks.Len())

	for i := 0; i < 3; i++ {
		_, err = ks.New()
		t.NoError(err)
	}

	// NOTE check in info
	uks, err := NewDeployKeyStorage(t.db)
	t.NoError(err)
	for i := range ks.keys {
		a := ks.keys[i]
		b := uks.keys[i]

		t.Equal(a.Key(), b.Key())
		t.True(localtime.Equal(a.AddedAt(), b.AddedAt()))
	}
}

func (t *testDeployKeyStorageWithDatabase) TestRevoke() {
	ks, err := NewDeployKeyStorage(t.db)
	t.NoError(err)
	t.Equal(0, ks.Len())

	for i := 0; i < 3; i++ {
		_, err = ks.New()
		t.NoError(err)
	}

	nk, err := ks.New()
	t.NoError(err)
	t.NoError(ks.Revoke(nk.Key()))

	// NOTE check in info
	uks, err := loadDeployKeys(t.db)
	t.NoError(err)
	for i := range ks.keys {
		a := ks.keys[i]
		b := uks[i]

		t.Equal(a.Key(), b.Key())
		t.True(localtime.Equal(a.AddedAt(), b.AddedAt()))
	}
}

func TestDeployKeyStorageWithMongodb(t *testing.T) {
	handler := new(testDeployKeyStorageWithDatabase)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}

type testDeployKeyEncode struct {
	suite.Suite
	enc encoder.Encoder
}

func (t *testDeployKeyEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.AddHinter(key.BTCPublickeyHinter))
}

func (t *testDeployKeyEncode) TestEncode() {
	dk := NewDeployKey()

	b, err := t.enc.Marshal(dk)
	t.NoError(err)

	var udk DeployKey
	t.NoError(t.enc.Unmarshal(b, &udk))

	t.Equal(dk.Key(), udk.Key())
	t.True(localtime.Equal(dk.AddedAt(), udk.AddedAt()))
}

func TestDeployKeyEncodeJSON(t *testing.T) {
	b := new(testDeployKeyEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestDeployKeyEncodeBSON(t *testing.T) {
	b := new(testDeployKeyEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
