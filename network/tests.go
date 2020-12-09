// +build test

package network

import (
	"bytes"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/spikeekips/mitum/util/logging"
)

//lint:ignore U1000 debugging inside test
var log logging.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	log = logging.NewLogger(&l, true)
}

func CompareNodeInfo(t *testing.T, a, b NodeInfo) {
	assert.True(t, a.Address().Equal(b.Address()))
	assert.True(t, a.Publickey().Equal(b.Publickey()))
	assert.True(t, a.NetworkID().Equal(b.NetworkID()))
	assert.Equal(t, a.State(), b.State())
	assert.Equal(t, a.Version(), b.Version())
	assert.Equal(t, a.URL(), b.URL())

	assert.Equal(t, a.LastBlock().Height(), b.LastBlock().Height())
	assert.Equal(t, a.LastBlock().Round(), b.LastBlock().Round())
	assert.True(t, a.LastBlock().Proposal().Equal(b.LastBlock().Proposal()))
	assert.True(t, a.LastBlock().PreviousBlock().Equal(b.LastBlock().PreviousBlock()))
	assert.True(t, a.LastBlock().OperationsHash().Equal(b.LastBlock().OperationsHash()))
	assert.True(t, a.LastBlock().StatesHash().Equal(b.LastBlock().StatesHash()))

	assert.Equal(t, a.Config(), b.Config())

	as := a.Nodes()
	bs := b.Nodes()
	assert.Equal(t, len(as), len(bs))

	sort.Slice(as, func(i, j int) bool {
		return bytes.Compare(as[i].Address().Bytes(), as[j].Address().Bytes()) < 0
	})
	sort.Slice(bs, func(i, j int) bool {
		return bytes.Compare(bs[i].Address().Bytes(), bs[j].Address().Bytes()) < 0
	})
	for i := range as {
		assert.True(t, as[i].Address().Equal(bs[i].Address()))
		assert.True(t, as[i].Publickey().Equal(bs[i].Publickey()))
	}
}
