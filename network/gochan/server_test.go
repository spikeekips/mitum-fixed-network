package channetwork

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/network"
)

type testChanSever struct {
	suite.Suite
}

func (t *testChanSever) TestNew() {
	s := NewServer(nil)

	t.Implements((*network.Server)(nil), s)
}

func TestChanSever(t *testing.T) {
	suite.Run(t, new(testChanSever))
}
