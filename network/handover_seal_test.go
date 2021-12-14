package network

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testHandoverSeal struct {
	suite.Suite
	local node.Local
}

func (t *testHandoverSeal) SetupSuite() {
	t.local = node.NewLocal(base.RandomStringAddress(), key.NewBasePrivatekey())
}

func (t *testHandoverSeal) TestIsValidSeal() {
	t.Run("valid", func() {
		sl, err := NewHandoverSealV0(StartHandoverSealV0Hint, t.local.Privatekey(), t.local.Address(), NewNilConnInfo("showme"), nil)
		t.NoError(err)

		t.NoError(IsValidHandoverSeal(t.local, sl, nil))
	})

	t.Run("empty conninfo", func() {
		sl, err := NewHandoverSealV0(StartHandoverSealV0Hint, t.local.Privatekey(), t.local.Address(), NewNilConnInfo("showme"), nil)
		t.NoError(err)

		sl.ci = nil
		err = IsValidHandoverSeal(t.local, sl, nil)
		t.True(errors.Is(err, isvalid.InvalidError))
	})

	t.Run("empty address", func() {
		sl, err := NewHandoverSealV0(StartHandoverSealV0Hint, t.local.Privatekey(), t.local.Address(), NewNilConnInfo("showme"), nil)
		t.NoError(err)

		sl.ad = nil
		err = IsValidHandoverSeal(t.local, sl, nil)
		t.True(errors.Is(err, isvalid.InvalidError))
	})

	t.Run("address not matched", func() {
		other := node.NewLocal(base.RandomStringAddress(), t.local.Privatekey())

		sl, err := NewHandoverSealV0(StartHandoverSealV0Hint, t.local.Privatekey(), other.Address(), NewNilConnInfo("showme"), nil)
		t.NoError(err)

		err = IsValidHandoverSeal(t.local, sl, nil)
		t.True(errors.Is(err, isvalid.InvalidError))
	})
}

func TestHandoverSeal(t *testing.T) {
	suite.Run(t, new(testHandoverSeal))
}
