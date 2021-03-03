package base

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var tinyFactHint = hint.MustHintWithType(hint.Type{0xff, 0xf4}, "0.1", "tiny-fact")

type tinyFact struct {
	A string
}

func (tf tinyFact) Hint() hint.Hint {
	return tinyFactHint
}

func (tf tinyFact) IsValid([]byte) error {
	if len(tf.A) < 1 {
		return isvalid.InvalidError.Errorf("empty A")
	}

	return nil
}

func (tf tinyFact) Hash() valuehash.Hash {
	return valuehash.NewSHA256(tf.Bytes())
}

func (tf tinyFact) Bytes() []byte {
	return []byte(tf.A)
}

func (tf tinyFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		A string
	}{
		HintedHead: jsonenc.NewHintedHead(tf.Hint()),
		A:          tf.A,
	})
}

func (tf tinyFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(struct {
		HI hint.Hint `bson:"_hint"`
		A  string
	}{
		HI: tf.Hint(),
		A:  tf.A,
	})
}

type testVoteproof struct {
	suite.Suite
	threshold Threshold
}

func (t *testVoteproof) SetupTest() {
	t.threshold, _ = NewThreshold(10, 40)
}

func (t *testVoteproof) TestInvalidHeight() {
	vp := VoteproofV0{height: Height(-3)}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidStage() {
	vp := VoteproofV0{stage: Stage(100)}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidThreshold() {
	vp := VoteproofV0{stage: StageINIT, thresholdRatio: ThresholdRatio(140)}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidResult() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultType(10),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyMajority() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       nil,
		finishedAt:     localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "empty majority")
}

func (t *testVoteproof) TestInvalidMajority() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       tinyFact{A: ""},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyFacts() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       tinyFact{A: "showme"},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyBallots() {
	fact := tinyFact{A: "showme"}

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []Fact{fact},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyVotes() {
	fact := tinyFact{A: "showme"}

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []Fact{fact},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestWrongVotesCount() {
	n0 := RandomNode("n0")

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []Fact{fact},
		votes: []VoteproofNodeFact{
			{
				address: n0.Address(),
				fact:    factHash,
			},
		},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidFactHash() {
	n0 := RandomNode("n0")
	fact := tinyFact{A: "showme"}

	invalidFactHash := valuehash.SHA256{}

	factSignature, _ := n0.Privatekey().Sign(invalidFactHash.Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []Fact{fact},
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          invalidFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, valuehash.EmptyHashError))
}

func (t *testVoteproof) TestUnknownFactHash() {
	n0 := RandomNode("n0")
	n1 := RandomNode("n1")
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	unknownFact := tinyFact{A: "killme"}
	unknownFactSignature, _ := n1.Privatekey().Sign(unknownFact.Hash().Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []Fact{fact},
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
			{
				address:       n1.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          unknownFact.Hash(),
				factSignature: unknownFactSignature,
				signer:        n1.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteproof) TestFactNotFound() {
	n0 := RandomNode("n0")

	fact := tinyFact{A: "showme"}

	newFact := tinyFact{A: "killme"}
	newFactHash := newFact.Hash()
	newFactSignature, _ := n0.Privatekey().Sign(newFactHash.Bytes())

	threshold, _ := NewThreshold(1, 40)
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []Fact{fact},
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          newFactHash,
				factSignature: newFactSignature,
				signer:        n0.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteproof) TestSuplusFacts() {
	n0 := RandomNode("n0")

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []Fact{fact, fact},
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteproof) TestNotYetButNot() {
	n0 := RandomNode("n0")

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), RandomStringAddress(), RandomStringAddress()},
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultDraw,
		majority:       fact,
		facts:          []Fact{fact},
		votes: []VoteproofNodeFact{
			{
				address:       n0.Address(),
				ballot:        valuehash.RandomSHA256(),
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result should be not-yet")
}

func (t *testVoteproof) TestDrawButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())

	fact1 := tinyFact{A: "fact1"}
	factHash1 := fact1.Hash()
	factSignature1, _ := n1.Privatekey().Sign(factHash1.Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), n1.Address()},
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact0,
		facts:          []Fact{fact0, fact1},
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
				fact:          factHash1,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), "DRAW")
}

func (t *testVoteproof) TestMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), n1.Address()},
		thresholdRatio: threshold.Ratio,
		result:         VoteResultDraw,
		facts:          []Fact{fact0},
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
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), " result=MAJORITY")
}

func TestVoteproof(t *testing.T) {
	suite.Run(t, new(testVoteproof))
}

func (t *testVoteproof) TestCompareVoteproof() {
	cases := []struct {
		name     string
		aHeight  Height
		aRound   Round
		aStage   Stage
		bHeight  Height
		bRound   Round
		bStage   Stage
		expected int
	}{
		{
			name:     "height higher",
			aHeight:  Height(3),
			aRound:   Round(0),
			aStage:   StageINIT,
			bHeight:  Height(2),
			bRound:   Round(10),
			bStage:   StageACCEPT,
			expected: 1,
		},
		{
			name:     "height lower",
			aHeight:  Height(3),
			aRound:   Round(0),
			aStage:   StageINIT,
			bHeight:  Height(9),
			bRound:   Round(10),
			bStage:   StageACCEPT,
			expected: -1,
		},
		{
			name:     "height same, round higher",
			aHeight:  Height(3),
			aRound:   Round(11),
			aStage:   StageINIT,
			bHeight:  Height(3),
			bRound:   Round(10),
			bStage:   StageACCEPT,
			expected: 1,
		},
		{
			name:     "height same, round lower",
			aHeight:  Height(3),
			aRound:   Round(9),
			aStage:   StageINIT,
			bHeight:  Height(3),
			bRound:   Round(10),
			bStage:   StageACCEPT,
			expected: -1,
		},
		{
			name:     "same height round, but stage higher",
			aHeight:  Height(3),
			aRound:   Round(9),
			aStage:   StageACCEPT,
			bHeight:  Height(3),
			bRound:   Round(9),
			bStage:   StageINIT,
			expected: 1,
		},
		{
			name:     "same height round, but stage lower",
			aHeight:  Height(3),
			aRound:   Round(9),
			aStage:   StageINIT,
			bHeight:  Height(3),
			bRound:   Round(9),
			bStage:   StageACCEPT,
			expected: -1,
		},
		{
			name:     "same height round and stage",
			aHeight:  Height(3),
			aRound:   Round(9),
			aStage:   StageINIT,
			bHeight:  Height(3),
			bRound:   Round(9),
			bStage:   StageINIT,
			expected: 0,
		},
	}

	newVoteproof := func(height Height, round Round, stage Stage) Voteproof {
		n0 := RandomNode("n0")
		fact := tinyFact{A: "showme"}
		factHash := fact.Hash()
		factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

		return VoteproofV0{
			height:         height,
			round:          round,
			stage:          stage,
			thresholdRatio: t.threshold.Ratio,
			result:         VoteResultMajority,
			majority:       fact,
			facts:          []Fact{fact},
			votes: []VoteproofNodeFact{
				{
					address:       n0.Address(),
					ballot:        valuehash.RandomSHA256(),
					fact:          factHash,
					factSignature: factSignature,
					signer:        n0.Publickey(),
				},
			},
			finishedAt: localtime.Now(),
		}
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				avp := newVoteproof(c.aHeight, c.aRound, c.aStage)
				bvp := newVoteproof(c.bHeight, c.bRound, c.bStage)
				result := CompareVoteproof(avp, bvp)

				t.Equal(c.expected, result, "%d: %v; %v != %v", i, c.name, c.expected, result)
			},
		)
	}
}
