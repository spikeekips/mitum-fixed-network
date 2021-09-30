package channetwork

import (
	"testing"

	"github.com/spikeekips/mitum/network"
	"github.com/stretchr/testify/suite"
)

type testChanServer struct {
	suite.Suite
}

func (t *testChanServer) TestNew() {
	s := NewServer(nil, nil)

	t.Implements((*network.Server)(nil), s)
}

func TestChanServer(t *testing.T) {
	suite.Run(t, new(testChanServer))
}
