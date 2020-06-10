package base

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/valuehash"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
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
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          map[valuehash.Hash]Fact{factHash: fact},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyVotes() {
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestWrongVotesCount() {
	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
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
		facts:          map[valuehash.Hash]Fact{invalidFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
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
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	unknownFactHash := valuehash.RandomSHA256()

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          map[valuehash.Hash]Fact{unknownFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
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
	factHash := fact.Hash()

	newFact := tinyFact{A: "killme"}
	newFactHash := newFact.Hash()
	newFactSignature, _ := n0.Privatekey().Sign(newFactHash.Bytes())

	threshold, _ := NewThreshold(1, 40)
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
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

func (t *testVoteproof) TestUnknownNodeFound() {
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	n0 := NewShortAddress("n0")

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0:                    valuehash.RandomSHA256(),
			NewShortAddress("n2"): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0: {
				fact: factHash,
			},
			NewShortAddress("n1"): {
				fact: factHash,
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "unknown node found")
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
		facts: map[valuehash.Hash]Fact{
			factHash:                 fact,
			valuehash.RandomSHA256(): fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
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
		suffrages:      []Address{n0.Address(), RandomShortAddress(), RandomShortAddress()},
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultDraw,
		majority:       fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
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
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
			factHash1: fact1,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): {
				address:       n1.Address(),
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
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				address:       n0.Address(),
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): {
				address:       n1.Address(),
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
