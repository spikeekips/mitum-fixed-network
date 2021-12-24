// +build test

package network

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testBlockdataFetchRemote struct {
	suite.Suite
}

func (t *testBlockdataFetchRemote) TestNew() {
	item := block.NewBaseBlockdataMapItem(block.BlockdataManifest, util.UUID().String(), "http://google.com")
	r, err := FetchBlockdataFromRemote(context.Background(), item)
	t.NoError(err)

	b, err := ioutil.ReadAll(r)
	t.NoError(err)
	t.NotNil(b)
}

func (t *testBlockdataFetchRemote) TestHTTPS() {
	item := block.NewBaseBlockdataMapItem(block.BlockdataManifest, util.UUID().String(), "https://google.com")
	_, err := FetchBlockdataFromRemote(context.Background(), item)
	t.NoError(err)
}

func (t *testBlockdataFetchRemote) TestUnknownScheme() {
	item := block.NewBaseBlockdataMapItem(block.BlockdataManifest, util.UUID().String(), "ftp://google.com")
	_, err := FetchBlockdataFromRemote(context.Background(), item)
	t.Contains(err.Error(), "not yet supported")
}

func TestBlockdataFetchRemote(t *testing.T) {
	suite.Run(t, new(testBlockdataFetchRemote))
}
