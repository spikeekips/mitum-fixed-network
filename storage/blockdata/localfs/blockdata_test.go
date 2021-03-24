package localfs

import (
	"os"
	"testing"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testBlockData struct {
	suite.Suite
	JSONEnc  *jsonenc.Encoder
	baseRoot string
	root     string
}

func (t *testBlockData) SetupSuite() {
	t.JSONEnc = jsonenc.NewEncoder()

	p, err := os.MkdirTemp("", "localfs-")
	if err != nil {
		panic(err)
	}

	t.baseRoot = p
}

func (t *testBlockData) SetupTest() {
	p, err := os.MkdirTemp(t.baseRoot, "localfs-")
	if err != nil {
		panic(err)
	}

	t.root = p
}

func (t *testBlockData) TearDownSuite() {
	_ = os.RemoveAll(t.baseRoot)
}

func (t *testBlockData) TestNew() {
	st := NewBlockData(t.root, t.JSONEnc)
	t.Implements((*blockdata.BlockData)(nil), st)
	t.NoError(st.Initialize())
}

func (t *testBlockData) TestRootDoesNotExist() {
	st := NewBlockData(util.UUID().String(), t.JSONEnc)
	err := st.Initialize()
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testBlockData) TestRemove() {
	st := NewBlockData(t.root, t.JSONEnc)
	t.NoError(st.Initialize())

	t.NoError(st.CreateDirectory(st.HeightDirectory(33, true)))
	found, removed, err := st.exists(33)
	t.NoError(err)
	t.True(found)
	t.False(removed)

	t.NoError(st.Remove(33))
	found, removed, err = st.exists(33)
	t.NoError(err)
	t.True(found)
	t.True(removed)

	found, err = st.Exists(33)
	t.NoError(err)
	t.False(found)
}

func (t *testBlockData) TestRemoveAll() {
	st := NewBlockData(t.root, t.JSONEnc)
	t.NoError(st.Initialize())

	t.NoError(st.CreateDirectory(st.HeightDirectory(33, true)))
	found, removed, err := st.exists(33)
	t.NoError(err)
	t.True(found)
	t.False(removed)

	t.NoError(st.RemoveAll(33))
	found, removed, err = st.exists(33)
	t.NoError(err)
	t.False(found)
	t.False(removed)

	found, err = st.Exists(33)
	t.NoError(err)
	t.False(found)
}

func TestBlockData(t *testing.T) {
	suite.Run(t, new(testBlockData))
}
