package ballot

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testBaseBallotV0 struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testBaseBallotV0) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBaseBallotV0) TestIsReadyToSign() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: base.RandomStringAddress(),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
			signer:   t.pk.Publickey(),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // invalid Node
		bb := BaseBallotV0{
			node: base.StringAddress(""), // empty Address
		}
		err := bb.IsReadyToSign(nil)
		t.Contains(err.Error(), "empty address")
	}
}

func (t *testBaseBallotV0) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: base.RandomStringAddress(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}

	{ // invalid Node
		bb := BaseBallotV0{
			node: base.StringAddress(""), // empty Address
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty address")
	}
}

func TestBaseBallotV0(t *testing.T) {
	suite.Run(t, new(testBaseBallotV0))
}
