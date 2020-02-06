package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type testVoteProofJSON struct {
	suite.Suite
}

func (t *testVoteProofJSON) SetupSuite() {
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
}

func (t *testVoteProofJSON) TestMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteProofV0{
		height:    Height(33),
		round:     Round(3),
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
		majority: fact0,
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0.Address(): VoteProofNodeFact{
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): VoteProofNodeFact{
				fact:          factHash0,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
	}
	t.NoError(vp.IsValid(nil))

	b, err := util.JSONMarshal(vp)
	t.NoError(err)
	t.NotNil(b)
}

func TestVoteProofJSON(t *testing.T) {
	suite.Run(t, new(testVoteProofJSON))
}
