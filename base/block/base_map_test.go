package block

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testBaseBlockDataMap struct {
	suite.Suite
}

func (t *testBaseBlockDataMap) TestNew() {
	bd := NewBaseBlockDataMap(TestBlockDataWriterHint, 33)
	t.Implements((*BlockDataMap)(nil), bd)
}

func TestBaseBlockDataMap(t *testing.T) {
	suite.Run(t, new(testBaseBlockDataMap))
}
