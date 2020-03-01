package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type testVoteproofJSON struct {
	suite.Suite

	hs *hint.Hintset
}

func (t *testVoteproofJSON) SetupSuite() {
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType((ShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(VoteproofV0{}.Hint().Type(), "voteproof")
	_ = hint.RegisterType(tinyFact{}.Hint().Type(), "tiny-fact")

	t.hs = hint.NewHintset()
	t.hs.Add(valuehash.SHA256{})
	t.hs.Add(ShortAddress(""))
	t.hs.Add(key.BTCPublickey{})
	t.hs.Add(tinyFact{})
}

func (t *testVoteproofJSON) TestMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteproofV0{
		height:    Height(33),
		round:     Round(3),
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteproofMajority,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
		majority: fact0,
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): {
				fact:          factHash0,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	t.NoError(vp.IsValid(nil))

	b, err := util.JSONMarshal(vp)
	t.NoError(err)
	t.NotNil(b)

	je := encoder.NewJSONEncoder()
	je.SetHintset(t.hs)

	var uvp VoteproofV0
	t.NoError(je.Decode(b, &uvp))

	t.Equal(vp.Height(), uvp.Height())
	t.Equal(vp.Round(), uvp.Round())
	t.Equal(vp.threshold, uvp.threshold)
	t.Equal(vp.Result(), uvp.Result())
	t.Equal(vp.Stage(), uvp.Stage())

	t.Equal(vp.Majority().Bytes(), uvp.Majority().Bytes())
	t.Equal(len(vp.facts), len(uvp.facts))
	for h, f := range vp.facts {
		t.Equal(f.Bytes(), uvp.facts[h].Bytes())
	}
	t.Equal(len(vp.ballots), len(uvp.ballots))
	for a, h := range vp.ballots {
		t.True(h.Equal(uvp.ballots[a]))
	}
	t.Equal(len(vp.votes), len(uvp.votes))
	for a, f := range vp.votes {
		u := uvp.votes[a]

		t.True(f.fact.Equal(u.fact))
		t.True(f.factSignature.Equal(u.factSignature))
		t.True(f.signer.Equal(u.signer))
	}
}

func TestVoteproofJSON(t *testing.T) {
	suite.Run(t, new(testVoteproofJSON))
}
