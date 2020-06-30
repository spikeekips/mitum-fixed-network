package base

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testVoteproofBSON struct {
	suite.Suite

	hs *hint.Hintset
}

func (t *testVoteproofBSON) SetupSuite() {
	t.hs = hint.NewHintset()
	t.hs.Add(valuehash.SHA256{})
	t.hs.Add(ShortAddress(""))
	t.hs.Add(key.BTCPublickey{})
	t.hs.Add(tinyFact{})
}

func (t *testVoteproofBSON) TestMarshal() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteproofV0{
		height:         Height(33),
		round:          Round(3),
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), n1.Address()},
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		facts:          []Fact{fact0},
		majority:       fact0,
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			{
				address:       n1.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash0,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	t.NoError(vp.IsValid(nil))

	b, err := bsonenc.Marshal(vp)
	t.NoError(err)
	t.NotNil(b)

	be := bsonenc.NewEncoder()
	be.SetHintset(t.hs)

	var uvp VoteproofV0
	t.NoError(be.Decode(b, &uvp))

	t.Equal(vp.Height(), uvp.Height())
	t.Equal(vp.Round(), uvp.Round())
	t.Equal(vp.thresholdRatio, uvp.thresholdRatio)
	t.Equal(vp.Result(), uvp.Result())
	t.Equal(vp.Stage(), uvp.Stage())

	t.Equal(vp.Majority().Bytes(), uvp.Majority().Bytes())
	t.Equal(len(vp.facts), len(uvp.facts))
	for _, f := range vp.facts {
		var fact Fact

		for _, uf := range uvp.facts {
			if f.Hash().Equal(uf.Hash()) {
				fact = f
				break
			}
		}

		t.Equal(f.Bytes(), fact.Bytes())
	}
	t.Equal(len(vp.votes), len(uvp.votes))
	for a, f := range vp.votes {
		u := uvp.votes[a]

		t.True(f.fact.Equal(u.fact))
		t.True(f.factSignature.Equal(u.factSignature))
		t.True(f.signer.Equal(u.signer))
	}
}

func TestVoteproofBSON(t *testing.T) {
	suite.Run(t, new(testVoteproofBSON))
}
