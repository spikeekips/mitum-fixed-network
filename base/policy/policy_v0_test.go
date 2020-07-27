package policy

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
)

type testPolicyV0 struct {
	suite.Suite
}

func (t *testPolicyV0) TestNew() {
	po := NewPolicyV0(base.ThresholdRatio(33), 3, 6, 9)
	t.NoError(po.IsValid(nil))

	t.Implements((*Policy)(nil), po)

	t.Equal(base.ThresholdRatio(33), po.ThresholdRatio())
	t.Equal(uint(3), po.NumberOfActingSuffrageNodes())
	t.Equal(uint(6), po.MaxOperationsInSeal())
	t.Equal(uint(9), po.MaxOperationsInProposal())
}

func (t *testPolicyV0) TestZeroNumberOfActingSuffrageNodes() {
	po := NewPolicyV0(base.ThresholdRatio(33), 0, 6, 9)
	err := po.IsValid(nil)
	t.Contains(err.Error(), "NumberOfActingSuffrageNodes must be over 0")
}

func (t *testPolicyV0) TestZeroMaxOperationsInSeal() {
	po := NewPolicyV0(base.ThresholdRatio(33), 3, 0, 9)
	err := po.IsValid(nil)
	t.Contains(err.Error(), "MaxOperationsInSeal must be over 0")
}

func (t *testPolicyV0) TestZeroMaxOperationsInProposal() {
	po := NewPolicyV0(base.ThresholdRatio(33), 3, 6, 0)
	err := po.IsValid(nil)
	t.Contains(err.Error(), "MaxOperationsInProposal must be over 0")
}

func TestPolicyV0(t *testing.T) {
	s := new(testPolicyV0)

	suite.Run(t, s)
}
