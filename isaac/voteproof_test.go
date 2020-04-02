package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
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
		return isvalid.InvalidError.Wrapf("empty A")
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
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		A string
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(tf.Hint()),
		A:                  tf.A,
	})
}

type testVoteproof struct {
	suite.Suite
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
	threshold, _ := NewThreshold(10, 140)
	vp := VoteproofV0{stage: StageINIT, threshold: threshold}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidResult() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultType(10),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyMajority() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteproofV0{
		stage:      StageINIT,
		threshold:  threshold,
		result:     VoteResultMajority,
		majority:   nil,
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "empty majority")
}

func (t *testVoteproof) TestInvalidMajority() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  tinyFact{A: ""},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyFacts() {
	threshold, _ := NewThreshold(10, 40)

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  tinyFact{A: "showme"},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyBallots() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{factHash: fact},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyVotes() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestWrongVotesCount() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				fact: factHash,
			},
		},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidFactHash() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}

	invalidFactHash := valuehash.SHA256{}

	factSignature, _ := n0.Privatekey().Sign(invalidFactHash.Bytes())

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{invalidFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
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
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	unknownFactHash := valuehash.RandomSHA256()

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{unknownFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
				fact:          unknownFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "does not match")
	t.Contains(err.Error(), "factHash")
}

func (t *testVoteproof) TestFactNotFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	n0 := NewShortAddress("n0")

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]operation.Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0: {
				fact: valuehash.RandomSHA256(),
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteproof) TestUnknownNodeFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	n0 := NewShortAddress("n0")

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact,
		facts: map[valuehash.Hash]operation.Fact{
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
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()

	n0 := NewShortAddress("n0")
	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact,
		facts: map[valuehash.Hash]operation.Fact{
			factHash:                 fact,
			valuehash.RandomSHA256(): fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0: {
				fact: factHash,
			},
		},
		finishedAt: localtime.Now(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteproof) TestNotYetButNot() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)

	fact := tinyFact{A: "showme"}
	factHash := fact.Hash()
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultDraw,
		majority:  fact,
		facts: map[valuehash.Hash]operation.Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteproofNodeFact{
			n0.Address(): {
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

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())

	fact1 := tinyFact{A: "fact1"}
	factHash1 := fact1.Hash()
	factSignature1, _ := n1.Privatekey().Sign(factHash1.Bytes())

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  fact0,
		facts: map[valuehash.Hash]operation.Fact{
			factHash0: fact0,
			factHash1: fact1,
		},
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

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0 := fact0.Hash()
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteproofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultDraw,
		facts: map[valuehash.Hash]operation.Fact{
			factHash0: fact0,
		},
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
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), " result=MAJORITY")
}

func TestVoteproof(t *testing.T) {
	suite.Run(t, new(testVoteproof))
}
