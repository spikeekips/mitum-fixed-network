package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type tinyFact struct {
	A string
}

func (tf tinyFact) IsValid([]byte) error {
	if len(tf.A) < 1 {
		return InvalidError.Wrapf("empty A")
	}

	return nil
}

func (tf tinyFact) Hash([]byte) (valuehash.Hash, error) {
	return valuehash.NewSHA256(tf.Bytes()), nil
}

func (tf tinyFact) Bytes() []byte {
	return []byte(tf.A)
}

type testVoteProof struct {
	suite.Suite
}

func (t *testVoteProof) TestInvalidHeight() {
	vp := VoteProofV0{height: Height(-3)}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestInvalidStage() {
	vp := VoteProofV0{stage: Stage(100)}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestInvalidThreshold() {
	threshold, _ := NewThreshold(10, 140)
	vp := VoteProofV0{stage: StageINIT, threshold: threshold}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestInvalidResult() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofResultType(10),
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestEmptyMajority() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  nil,
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
	t.Contains(err.Error(), "empty majority")
}

func (t *testVoteProof) TestInvalidMajority() {
	threshold, _ := NewThreshold(10, 40)
	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofNotYet,
		majority:  tinyFact{A: ""},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestEmptyFacts() {
	threshold, _ := NewThreshold(10, 40)

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofNotYet,
		majority:  tinyFact{A: "showme"},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestEmptyBallots() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
	}
	err = vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestEmptyVotes() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			NewShortAddress("n0"): valuehash.RandomSHA256(),
		},
	}
	err = vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestWrongVotesCount() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofNotYet,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
			n1.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0.Address(): VoteProofNodeFact{
				fact: factHash,
			},
		},
	}
	err = vp.IsValid(nil)
	t.True(xerrors.Is(err, InvalidError))
}

func (t *testVoteProof) TestInvalidFactHash() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}

	invalidFactHash := valuehash.SHA256{}

	factSignature, _ := n0.Privatekey().Sign(invalidFactHash.Bytes())

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{invalidFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0.Address(): VoteProofNodeFact{
				fact:          invalidFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err := vp.IsValid(nil)
	t.True(xerrors.Is(err, valuehash.EmptyHashError))
}

func (t *testVoteProof) TestUnknownFactHash() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	unknownFactHash := valuehash.RandomSHA256()

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{unknownFactHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0.Address(): VoteProofNodeFact{
				fact:          unknownFactHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err = vp.IsValid(nil)
	t.Contains(err.Error(), "does not match")
	t.Contains(err.Error(), "factHash")
}

func (t *testVoteProof) TestFactNotFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	n0 := NewShortAddress("n0")

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact,
		facts:     map[valuehash.Hash]Fact{factHash: fact},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0: VoteProofNodeFact{
				fact: valuehash.RandomSHA256(),
			},
		},
	}
	err = vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteProof) TestUnknownNodeFound() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	n0 := NewShortAddress("n0")

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0:                    valuehash.RandomSHA256(),
			NewShortAddress("n2"): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0: VoteProofNodeFact{
				fact: factHash,
			},
			NewShortAddress("n1"): VoteProofNodeFact{
				fact: factHash,
			},
		},
	}
	err = vp.IsValid(nil)
	t.Contains(err.Error(), "unknown node found")
}

func (t *testVoteProof) TestSuplusFacts() {
	threshold, _ := NewThreshold(10, 40)
	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)

	n0 := NewShortAddress("n0")
	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash:                 fact,
			valuehash.RandomSHA256(): fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0: valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0: VoteProofNodeFact{
				fact: factHash,
			},
		},
	}
	err = vp.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteProof) TestNotYetButNot() {
	threshold, _ := NewThreshold(10, 40)

	n0 := RandomLocalNode("n0", nil)

	fact := tinyFact{A: "showme"}
	factHash, err := fact.Hash(nil)
	t.NoError(err)
	factSignature, _ := n0.Privatekey().Sign(factHash.Bytes())

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofDraw,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: map[Address]valuehash.Hash{
			n0.Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			n0.Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: factSignature,
				signer:        n0.Publickey(),
			},
		},
	}
	err = vp.IsValid(nil)
	t.Contains(err.Error(), "result should be not-yet")
}

func (t *testVoteProof) TestDrawButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())

	fact1 := tinyFact{A: "fact1"}
	factHash1, _ := fact1.Hash(nil)
	factSignature1, _ := n1.Privatekey().Sign(factHash1.Bytes())

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofMajority,
		majority:  fact0,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
			factHash1: fact1,
		},
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
				fact:          factHash1,
				factSignature: factSignature1,
				signer:        n1.Publickey(),
			},
		},
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), "DRAW")
}

func (t *testVoteProof) TestMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomLocalNode("n0", nil)
	n1 := RandomLocalNode("n1", nil)

	fact0 := tinyFact{A: "fact0"}
	factHash0, _ := fact0.Hash(nil)
	factSignature0, _ := n0.Privatekey().Sign(factHash0.Bytes())
	factSignature1, _ := n1.Privatekey().Sign(factHash0.Bytes())

	vp := VoteProofV0{
		stage:     StageINIT,
		threshold: threshold,
		result:    VoteProofDraw,
		facts: map[valuehash.Hash]Fact{
			factHash0: fact0,
		},
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
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), " result=MAJORITY")
}

func TestVoteProof(t *testing.T) {
	suite.Run(t, new(testVoteProof))
}
