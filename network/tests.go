// +build test

package network

import (
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/spikeekips/mitum/util/logging"
)

var log logging.Logger // nolint

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
}
