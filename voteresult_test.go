package mitum

import (
	"testing"

	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type tinyFact struct {
	A string
}

func (tf tinyFact) IsValid(b []byte) error {
	if len(tf.A) < 1 {
		return InvalidError.Wrapf("empty A")
	}

	return nil
}

func (tf tinyFact) Hash(b []byte) (valuehash.Hash, error) {
	return valuehash.NewSHA256(tf.Bytes()), nil
}

func (tf tinyFact) Bytes() []byte {
	return []byte(tf.A)
}

type testVoteResult struct {
	suite.Suite
}

func (t *testVoteResult) TestInvalidHeight() {
	vr := VoteResult{height: Height(-3)}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestInvalidStage() {
	vr := VoteResult{stage: Stage(100)}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestInvalidThreshold() {
	threshold, _ := NewThreshold(10, 140)
	vr := VoteResult{stage: StageINIT, threshold: threshold}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestInvalidResult() {
	threshold, _ := NewThreshold(10, 40)
	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultType(10),
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestEmptyMajority() {
	threshold, _ := NewThreshold(10, 40)
	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultMajority,
		majority:  nil,
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
	t.Contains(err.Error(), "empty majority")
}

func (t *testVoteResult) TestInvalidMajority() {
	threshold, _ := NewThreshold(10, 40)
	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  tinyFact{A: ""},
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestEmptyFacts() {
	threshold, _ := NewThreshold(10, 40)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  tinyFact{A: "showme"},
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestEmptyBallots() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
	}
	err = vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestEmptyVotes() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
	}
	err = vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestWrongVotesCount() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
			NewShortAddress("n1"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash,
		},
	}
	err = vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestInvalidFactHash() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{valuehash.SHA512{}: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash,
		},
	}
	err = vr.IsValid(nil)
	t.True(xerrors.Is(err, valuehash.EmptyHashError))
}

func (t *testVoteResult) TestFactNotFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteResult) TestSuplusFacts() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash:                 fact,
			valuehash.RandomSHA256(): fact,
		},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash,
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteResult) TesteNotYetButNot() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultDraw,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash,
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "result should be not-yet")
}

func (t *testVoteResult) TesteDrawButNot() {
	threshold, _ := NewThreshold(2, 80)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	fact1 := tinyFact{A: "fact1"}
	factHash1, _ := fact1.Hash(nil)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultNotYet,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
			factHash1: fact1,
		},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
			NewShortAddress("n1"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash0,
			NewShortAddress("n1"): factHash1,
		},
	}
	err := vr.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), "RAW")
}

func (t *testVoteResult) TesteMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)

	vr := VoteResult{
		stage:     StageSIGN,
		threshold: threshold,
		result:    VoteResultDraw,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
			NewShortAddress("n1"): valuehash.RandomSHA256(),
		},
		votes: map[Address]valuehash.Hash{
			NewShortAddress("n0"): factHash0,
			NewShortAddress("n1"): factHash0,
		},
	}
	err := vr.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), " result=MAJORITY")
}

func TestVoteResult(t *testing.T) {
	suite.Run(t, new(testVoteResult))
}
