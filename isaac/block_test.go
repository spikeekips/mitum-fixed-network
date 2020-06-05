package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testBlock struct {
	baseTestSyncer
}

func (t *testBlock) TestBlockIsValid() {
	localstate := t.localstates(1)[0]
	blk, found, err := localstate.Storage().BlockByHeight(2)
	t.NoError(err)
	t.True(found)

	orig := localstate.Policy().NetworkID()
	t.NoError(blk.IsValid(orig))

	n := []byte(util.UUID().String())

	err = blk.IsValid(n)
	t.True(xerrors.Is(err, key.SignatureVerificationFailedError)) // with invalid network id
}

func TestBlock(t *testing.T) {
	suite.Run(t, new(testBlock))
}
