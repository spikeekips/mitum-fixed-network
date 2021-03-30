package channetwork

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/network"
)

type testChanServer struct {
	suite.Suite
}

func (t *testChanServer) TestNew() {
	s := NewServer(nil)

	t.Implements((*network.Server)(nil), s)
}

func TestChanServer(t *testing.T) {
	suite.Run(t, new(testChanServer))
}
