package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
)

type testBaseBallotV0 struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBaseBallotV0) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBaseBallotV0) TestIsReadyToSign() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: NewShortAddress("test-for-init-ballot"),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
			signer:   t.pk.Publickey(),
		}
		t.NoError(bb.IsReadyToSign(nil))
	}

	{ // invalid Node
		bb := BaseBallotV0{
			node: NewShortAddress(""), // empty Address
		}
		err := bb.IsReadyToSign(nil)
		t.Contains(err.Error(), "empty address")
	}
}

func (t *testBaseBallotV0) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: NewShortAddress("test-for-init-ballot"),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}

	{ // invalid Node
		bb := BaseBallotV0{
			node: NewShortAddress(""), // empty Address
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty address")
	}
}

func TestBaseBallotV0(t *testing.T) {
	suite.Run(t, new(testBaseBallotV0))
}
