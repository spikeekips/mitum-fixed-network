package network

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type testChannelNetwork struct {
	suite.Suite
}

func (t *testChannelNetwork) TestNew() {
	home := node.NewRandomHome()
	cn := NewChannelNetwork(home)
	t.Equal(1, len(cn.chans))
}

func (t *testChannelNetwork) TestReceive() {
	defer common.DebugPanic()

	home := node.NewRandomHome()
	cn := NewChannelNetwork(home)

	var networks []*ChannelNetwork
	for i := 0; i < 4; i++ {
		c := NewChannelNetwork(node.NewRandomHome())
		_ = cn.AddMembers(c)
		networks = append(networks, c)

		t.NoError(c.Start())
	}

	t.NoError(cn.Start())

	// create 4 networks
	var wg sync.WaitGroup
	wg.Add(len(networks))

	sl, _ := seal.NewSealBodySigned(home.PrivateKey(), "a", 10)

	err := cn.Broadcast(sl)
	t.NoError(err)

	for _, n := range networks {
		m := <-n.Reader()
		receivedSeal, ok := m.(seal.Seal)
		t.True(ok)

		t.True(sl.Equal(receivedSeal))
		wg.Done()
	}

	wg.Wait()
	t.NoError(cn.Stop())
}

func TestChannelNetwork(t *testing.T) {
	suite.Run(t, new(testChannelNetwork))
}
