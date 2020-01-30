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
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultType(10),
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestEmptyMajority() {
	threshold, _ := NewThreshold(10, 40)
	vr := VoteResult{
		stage:     StageINIT,
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
		stage:     StageINIT,
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
		stage:     StageINIT,
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
		stage:     StageINIT,
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
		stage:     StageINIT,
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

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact: factHash,
			},
		},
	}
	err = vr.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteResult) TestInvalidFactHash() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}

	invalidFactHash := valuehash.SHA256{}

	factSignature, _ := n0.Privatekey().Sign(invalidFactHash.Bytes())

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{invalidFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact:          invalidFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err := vr.IsValid(nil)
	t.True(xerrors.Is(err, valuehash.EmptyHashError))
}

func (t *testVoteResult) TestUnknownFactHash() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	unknownFactHash := valuehash.RandomSHA256()

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{unknownFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact:          unknownFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "does not match")
	t.Contains(err.Error(), "factHash")
}

func (t *testVoteResult) TestFactNotFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	n0 := NewShortAddress("n0")

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0: VoteResultNodeFact{
				fact: valuehash.RandomSHA256(),
			},
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

	n0 := NewShortAddress("n0")
	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash:                 fact,
			valuehash.RandomSHA256(): fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0: VoteResultNodeFact{
				fact: factHash,
			},
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteResult) TesteNotYetButNot() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)

	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultDraw,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err = vr.IsValid(nil)
	t.Contains(err.Error(), "result should be not-yet")
}

func (t *testVoteResult) TesteDrawButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())

	fact1 := tinyFact{A: "fact1"}
	factHash1, _ := fact1.Hash(nil)
	factSignature1, _ := n1.Privatekey().Sign(factHash1.Bytes())

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultNotYet,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
			factHash1: fact1,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): VoteResultNodeFact{
				fact:          factHash1,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
	}
	err := vr.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), "RAW")
}

func (t *testVoteResult) TesteMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vr := VoteResult{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteResultDraw,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes0: map[Address]VoteResultNodeFact{
			n0.Address(): VoteResultNodeFact{
				fact:          factHash0,
				factSignature: factSignature0,
				signer:        n0.Publickey(),
			},
			n1.Address(): VoteResultNodeFact{
				fact:          factHash0,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
	}
	err := vr.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), " result=MAJORITY")
}

func TestVoteResult(t *testing.T) {
	suite.Run(t, new(testVoteResult))
}
