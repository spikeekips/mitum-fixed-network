//go:build test
// +build test

package network

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CompareNodeInfo(t *testing.T, a, b NodeInfo) {
	assert.True(t, a.Hint().Equal(b.Hint()))
	assert.True(t, a.Address().Equal(b.Address()))
	assert.True(t, a.Publickey().Equal(b.Publickey()))
	assert.True(t, a.NetworkID().Equal(b.NetworkID()))
	assert.Equal(t, a.State(), b.State())
	assert.Equal(t, a.Version(), b.Version())

	assert.Equal(t, a.LastBlock().Height(), b.LastBlock().Height())
	assert.Equal(t, a.LastBlock().Round(), b.LastBlock().Round())
	assert.True(t, a.LastBlock().Proposal().Equal(b.LastBlock().Proposal()))
	assert.True(t, a.LastBlock().PreviousBlock().Equal(b.LastBlock().PreviousBlock()))
	assert.True(t, a.LastBlock().OperationsHash().Equal(b.LastBlock().OperationsHash()))
	assert.True(t, a.LastBlock().StatesHash().Equal(b.LastBlock().StatesHash()))

	assert.Equal(t, a.Policy(), b.Policy())

	as := a.Nodes()
	bs := b.Nodes()
	assert.Equal(t, len(as), len(bs))

	sort.Slice(as, func(i, j int) bool {
		return bytes.Compare(as[i].Address.Bytes(), as[j].Address.Bytes()) < 0
	})
	sort.Slice(bs, func(i, j int) bool {
		return bytes.Compare(bs[i].Address.Bytes(), bs[j].Address.Bytes()) < 0
	})
	for i := range as {
		assert.True(t, as[i].Address.Equal(bs[i].Address))
		assert.True(t, as[i].Publickey.Equal(bs[i].Publickey))
		assert.True(t, as[i].ConnInfo().Equal(bs[i].ConnInfo()))
	}

	assert.True(t, a.ConnInfo().Equal(b.ConnInfo()))
}

func NilConnInfoChannel(s string) *DummyChannel {
	return NewDummyChannel(NewNilConnInfo(s))
}
