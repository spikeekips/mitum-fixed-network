package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testStateBootingHandler struct {
	baseTestStateHandler
}

func (t *testStateBootingHandler) TestNew() {
}

func TestStateBootingHandler(t *testing.T) {
	suite.Run(t, new(testStateBootingHandler))
}
