package deploy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testSetBlockDataMaps struct {
	isaac.BaseTest
	local *isaac.Local
	bc    *BlockDataCleaner
}

func (t *testSetBlockDataMaps) SetupTest() {
	t.BaseTest.SetupTest()
	t.local = t.Locals(1)[0]
	t.bc = NewBlockDataCleaner(t.local.BlockData().(*localfs.BlockData), DefaultTimeAfterRemoveBlockDataFiles)
}

func (t *testSetBlockDataMaps) remoteMap(bdm block.BlockDataMap) block.BlockDataMap {
	ubdm := bdm.(block.BaseBlockDataMap)

	items := ubdm.Items()
	for i := range items {
		item := items[i]
		j, err := ubdm.SetItem(item.SetURL(fmt.Sprintf("https://1.2.3.4/%s", item.URL()[8:])))
		t.NoError(err)

		ubdm = j
	}

	i, err := ubdm.UpdateHash()
	t.NoError(err)
	ubdm = i

	t.False(ubdm.IsLocal())

	return ubdm
}

func (t *testSetBlockDataMaps) TestWithoutMap() {
	handler := NewSetBlockDataMapsHandler(t.JSONEnc, t.local.Database(), t.bc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusBadRequest, res.StatusCode)

	t.Equal(network.ProblemMimetype, res.Header.Get("Content-Type"))
	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)
	t.Contains(pr.Title(), "failed to load blockdatamaps")
}

func (t *testSetBlockDataMaps) TestNew() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	bdm, found, err := t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	var files []string
	{
		items := bdm.(block.BaseBlockDataMap).Items()
		for i := range items {
			files = append(files, items[i].URL()[7:])
		}
	}

	nbdm := t.remoteMap(bdm)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	b, err := t.JSONEnc.Marshal([]block.BlockDataMap{nbdm})
	t.NoError(err)

	bf := bytes.NewBuffer(b)
	r.Body = io.NopCloser(bf)

	handler := NewSetBlockDataMapsHandler(t.JSONEnc, t.local.Database(), t.bc)
	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	ubdm, found, err := t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	aitems := nbdm.(block.BaseBlockDataMap).Items()
	bitems := ubdm.(block.BaseBlockDataMap).Items()

	for i := range aitems {
		a := aitems[i]
		b := bitems[i]

		t.Equal(a.Bytes(), b.Bytes())
	}

	t.False(t.local.BlockData().Exists(m.Height()))

	// NOTE files will not be removed pysically, just height directory will be
	// marked by .removed file.
	fs := t.local.BlockData().FS()
	for i := range files {
		_, err := fs.Open(files[i])
		t.NoError(err)
	}

	removed, err := fs.Open(filepath.Join(
		filepath.Dir(files[0]),
		localfs.BlockDirectoryRemovedTouchFile,
	))
	t.NoError(err)
	t.NotNil(removed)
}

func (t *testSetBlockDataMaps) TestStillLocalMap() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	bdm, found, err := t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	var files []string
	{
		items := bdm.(block.BaseBlockDataMap).Items()
		for i := range items {
			files = append(files, items[i].URL()[7:])
		}
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	b, err := t.JSONEnc.Marshal([]block.BlockDataMap{bdm})
	t.NoError(err)

	bf := bytes.NewBuffer(b)
	r.Body = io.NopCloser(bf)

	handler := NewSetBlockDataMapsHandler(t.JSONEnc, t.local.Database(), t.bc)
	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusOK, res.StatusCode)

	_, found, err = t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	t.True(t.local.BlockData().Exists(m.Height()))

	// NOTE local map will not be removed
	fs := t.local.BlockData().FS()
	for i := range files {
		_, err := fs.Open(files[i])
		t.NoError(err)
	}
}

func (t *testSetBlockDataMaps) TestInvalidMapHash() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	bdm, found, err := t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	nbdm := t.remoteMap(bdm)
	nbdm = nbdm.(block.BaseBlockDataMap).SetHash(valuehash.RandomSHA256())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	b, err := t.JSONEnc.Marshal([]block.BlockDataMap{nbdm})
	t.NoError(err)

	bf := bytes.NewBuffer(b)
	r.Body = io.NopCloser(bf)

	handler := NewSetBlockDataMapsHandler(t.JSONEnc, t.local.Database(), t.bc)
	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusBadRequest, res.StatusCode)

	t.Equal(network.ProblemMimetype, res.Header.Get("Content-Type"))
	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)

	t.Contains(pr.Title(), "invalid: incorrect block data map hash")
}

func (t *testSetBlockDataMaps) TestInvalidBlockHashMap() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	bdm, found, err := t.local.Database().BlockDataMap(m.Height())
	t.NoError(err)
	t.True(found)

	nbdm := t.remoteMap(bdm)
	nbdm = nbdm.(block.BaseBlockDataMap).SetBlock(valuehash.RandomSHA256())
	i, err := nbdm.(block.BaseBlockDataMap).UpdateHash()
	t.NoError(err)
	nbdm = i

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	b, err := t.JSONEnc.Marshal([]block.BlockDataMap{nbdm})
	t.NoError(err)

	bf := bytes.NewBuffer(b)
	r.Body = io.NopCloser(bf)

	handler := NewSetBlockDataMapsHandler(t.JSONEnc, t.local.Database(), t.bc)
	handler(w, r)

	res := w.Result()
	t.Equal(http.StatusBadRequest, res.StatusCode)

	t.Equal(network.ProblemMimetype, res.Header.Get("Content-Type"))
	pr, err := network.LoadProblemFromResponse(res)
	t.NoError(err)

	t.Contains(pr.Title(), "block hash does not match with manifest")
}

func TestSetBlockDataMaps(t *testing.T) {
	suite.Run(t, new(testSetBlockDataMaps))
}
