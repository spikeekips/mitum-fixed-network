package localfs

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testBlockdata struct {
	suite.Suite
	JSONEnc  *jsonenc.Encoder
	baseRoot string
	root     string
}

func (t *testBlockdata) SetupSuite() {
	t.JSONEnc = jsonenc.NewEncoder()

	p, err := os.MkdirTemp("", "localfs-")
	if err != nil {
		panic(err)
	}

	t.baseRoot = p
}

func (t *testBlockdata) SetupTest() {
	p, err := os.MkdirTemp(t.baseRoot, "localfs-")
	if err != nil {
		panic(err)
	}

	t.root = p
}

func (t *testBlockdata) TearDownSuite() {
	_ = os.RemoveAll(t.baseRoot)
}

func (t *testBlockdata) TestNew() {
	st := NewBlockdata(t.root, t.JSONEnc)
	t.Implements((*blockdata.Blockdata)(nil), st)
	t.NoError(st.Initialize())
}

func (t *testBlockdata) TestRootDoesNotExist() {
	st := NewBlockdata(util.UUID().String(), t.JSONEnc)
	err := st.Initialize()
	t.True(errors.Is(err, util.NotFoundError))
}

func (t *testBlockdata) TestRemove() {
	st := NewBlockdata(t.root, t.JSONEnc)
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

func (t *testBlockdata) TestRemoveAll() {
	st := NewBlockdata(t.root, t.JSONEnc)
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

func TestBlockdata(t *testing.T) {
	suite.Run(t, new(testBlockdata))
}
