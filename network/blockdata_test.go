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

type testBlockDataFetchRemote struct {
	suite.Suite
}

func (t *testBlockDataFetchRemote) TestNew() {
	item := block.NewBaseBlockDataMapItem(block.BlockDataManifest, util.UUID().String(), "http://google.com")
	r, err := FetchBlockDataFromRemote(context.Background(), item)
	t.NoError(err)

	b, err := ioutil.ReadAll(r)
	t.NoError(err)
	t.NotNil(b)
}

func (t *testBlockDataFetchRemote) TestHTTPS() {
	item := block.NewBaseBlockDataMapItem(block.BlockDataManifest, util.UUID().String(), "https://google.com")
	_, err := FetchBlockDataFromRemote(context.Background(), item)
	t.NoError(err)
}

func (t *testBlockDataFetchRemote) TestUnknownScheme() {
	item := block.NewBaseBlockDataMapItem(block.BlockDataManifest, util.UUID().String(), "ftp://google.com")
	_, err := FetchBlockDataFromRemote(context.Background(), item)
	t.Contains(err.Error(), "not yet supported")
}

func TestBlockDataFetchRemote(t *testing.T) {
	suite.Run(t, new(testBlockDataFetchRemote))
}
