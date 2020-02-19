package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testConsensusStateBootingHandler struct {
	baseTestConsensusStateHandler
}

func (t *testConsensusStateBootingHandler) TestNew() {
}

func TestConsensusStateBootingHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateBootingHandler))
}
