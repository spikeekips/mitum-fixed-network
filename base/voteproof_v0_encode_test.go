package base

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testVoteproofEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testVoteproofEncode) SetupSuite() {
	t.enc.Add(StringAddressHinter)
	t.enc.Add(key.BasePublickey{})
	t.enc.Add(DummyBallotFact{})
	t.enc.Add(VoteproofV0Hinter)
	t.enc.Add(SignedBallotFactHinter)
	t.enc.Add(BaseFactSignHinter)
	t.enc.Add(BallotFactSignHinter)
}

func (t *testVoteproofEncode) TestMarshal() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact := NewDummyBallotFact()
	fact.S = StageINIT
	fact.HT = Height(33)
	fact.R = Round(3)

	fs0, _ := NewBaseBallotFactSignFromFact(fact, n0.Address(), n0.Privatekey(), nil)
	fs1, _ := NewBaseBallotFactSignFromFact(fact, n1.Address(), n1.Privatekey(), nil)

	i := NewVoteproofV0(
		Height(33),
		Round(3),
		[]Address{n0.Address(), n1.Address()},
		threshold.Ratio,
		StageINIT,
	)

	i.BaseHinter = hint.NewBaseHinter(hint.NewHint(VoteproofV0Type, "v0.0.9"))

	vp := &i
	vp = vp.SetResult(VoteResultMajority).
		SetFacts([]BallotFact{fact}).
		SetMajority(fact).
		SetVotes([]SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs0),
			NewBaseSignedBallotFact(fact, fs1),
		}).
		Finish()
	t.NoError(vp.IsValid(nil))

	b, err := t.enc.Marshal(vp)
	t.NoError(err)
	t.NotNil(b)

	var uvp VoteproofV0
	t.NoError(encoder.Decode(b, t.enc, &uvp))

	t.True(vp.Hint().Equal(uvp.Hint()))
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

		af := f.Fact()
		bf := u.Fact()
		afs := f.FactSign()
		bfs := u.FactSign()

		t.True(af.Hash().Equal(bf.Hash()))
		t.Equal(af.Stage(), bf.Stage())
		t.Equal(af.Height(), bf.Height())
		t.Equal(af.Round(), bf.Round())

		t.True(afs.Node().Equal(bfs.Node()))
		t.True(afs.Signature().Equal(bfs.Signature()))
		t.True(afs.Signer().Equal(bfs.Signer()))
		t.True(localtime.Equal(afs.SignedAt(), bfs.SignedAt()))
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
