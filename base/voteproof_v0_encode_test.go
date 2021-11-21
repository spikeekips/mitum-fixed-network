package base

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testVoteproofEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testVoteproofEncode) SetupSuite() {
	t.enc.Add(StringAddress(""))
	t.enc.Add(key.BTCPublickeyHinter)
	t.enc.Add(tinyFact{})
	t.enc.Add(VoteproofV0{})
	t.enc.Add(BaseVoteproofNodeFact{})
}

func (t *testVoteproofEncode) TestMarshal() {
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
			BaseVoteproofNodeFact{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			BaseVoteproofNodeFact{
				address:       n1.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash0,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
		finishedAt: localtime.UTCNow(),
	}
	t.NoError(vp.IsValid(nil))

	b, err := t.enc.Marshal(vp)
	t.NoError(err)
	t.NotNil(b)

	hinter, err := DecodeVoteproof(b, t.enc)
	t.NoError(err)

	uvp, ok := hinter.(VoteproofV0)
	t.True(ok)

	t.Equal(vp.Height(), uvp.Height())
	t.Equal(vp.Round(), uvp.Round())
	t.Equal(vp.thresholdRatio, uvp.thresholdRatio)
	t.Equal(vp.Result(), uvp.Result())
	t.Equal(vp.Stage(), uvp.Stage())

	t.True(vp.Majority().Hash().Equal(uvp.Majority().Hash()))
	t.Equal(len(vp.facts), len(uvp.facts))
	for _, f := range vp.facts {
		var found bool
		for _, uf := range uvp.facts {
			if f.Hash().Equal(uf.Hash()) {
				found = true
				break
			}
		}

		t.True(found)
	}
	t.Equal(len(vp.votes), len(uvp.votes))
	for a, f := range vp.votes {
		u := uvp.votes[a]

		t.True(f.Fact().Equal(u.Fact()))
		t.True(f.Signature().Equal(u.Signature()))
		t.True(f.Signer().Equal(u.Signer()))
	}
}

func TestVoteproofEncodeJSON(t *testing.T) {
	b := new(testVoteproofEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestVoteproofEncodeBSON(t *testing.T) {
	b := new(testVoteproofEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
