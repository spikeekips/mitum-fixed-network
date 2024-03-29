package isaac

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testBlock struct {
	BaseTest
}

func (t *testBlock) TestBlockIsValid() {
	local := t.Locals(1)[0]

	bs := local.Blockdata().(*localfs.Blockdata)
	_, blk, err := localfs.LoadBlock(bs, 2)
	t.NoError(err)
	t.NotNil(blk)

	orig := local.Policy().NetworkID()
	t.NoError(blk.IsValid(orig))

	err = blk.IsValid(util.UUID().Bytes())
	t.True(errors.Is(err, key.SignatureVerificationFailedError)) // with invalid network id
}

func TestBlock(t *testing.T) {
	suite.Run(t, new(testBlock))
}
