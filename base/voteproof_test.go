package base

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

var tinyFactHint = hint.NewHint(hint.Type("tiny-fact"), "v0.1")

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

func (tf tinyFact) Stage() Stage {
	return StageINIT
}

func (tf tinyFact) Height() Height {
	return GenesisHeight
}

func (tf tinyFact) Round() Round {
	return Round(0)
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

var (
	DummyNodeType = hint.Type("dummy-node")
	DummyNodeHint = hint.NewHint(DummyNodeType, "v0.0.1")
)

type DummyNode struct {
	address    Address
	privatekey key.Privatekey
	publickey  key.Publickey
}

func NewDummyNode(address Address, privatekey key.Privatekey) *DummyNode {
	return &DummyNode{
		address:    address,
		privatekey: privatekey,
		publickey:  privatekey.Publickey(),
	}
}

func (ln *DummyNode) Hint() hint.Hint {
	return DummyNodeHint
}

func (ln *DummyNode) String() string {
	return ln.address.String()
}

func (ln *DummyNode) IsValid([]byte) error {
	return isvalid.Check(nil, false, ln.address, ln.publickey)
}

func (ln *DummyNode) Bytes() []byte {
	return util.ConcatBytesSlice(ln.address.Bytes(), ln.publickey.Bytes())
}

func (ln *DummyNode) Address() Address {
	return ln.address
}

func (ln *DummyNode) Privatekey() key.Privatekey {
	return ln.privatekey
}

func (ln *DummyNode) Publickey() key.Publickey {
	return ln.publickey
}

func RandomNode(name string) *DummyNode {
	return NewDummyNode(MustNewStringAddress(fmt.Sprintf("n-%s", name)), key.NewBasePrivatekey())
}

type testVoteproof struct {
	suite.Suite
	threshold Threshold
	pk        key.Privatekey
}

func (t *testVoteproof) SetupTest() {
	t.threshold, _ = NewThreshold(10, 40)
	t.pk = key.NewBasePrivatekey()
}

func (t *testVoteproof) signFact(n Address, priv key.Privatekey, fact BallotFact, networkID NetworkID) BallotFactSign {
	fs, _ := NewBaseBallotFactSignFromFact(fact, n, priv, networkID)

	return fs
}

func (t *testVoteproof) TestInvalidHeight() {
	vp := VoteproofV0{height: Height(-3)}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidStage() {
	vp := VoteproofV0{stage: Stage(100)}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidThreshold() {
	vp := VoteproofV0{stage: StageINIT, thresholdRatio: ThresholdRatio(140)}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidResult() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultType(10),
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyMajority() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       nil,
		finishedAt:     localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
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
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyFacts() {
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       tinyFact{A: "showme"},
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyBallots() {
	fact := tinyFact{A: "showme"}

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []BallotFact{fact},
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestEmptyVotes() {
	fact := tinyFact{A: "showme"}

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []BallotFact{fact},
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestWrongVotesCount() {
	n0 := RandomNode("n0")

	fact := NewDummyBallotFact()
	fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultNotYet,
		majority:       fact,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs),
		},
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testVoteproof) TestInvalidFactHash() {
	n0 := RandomNode("n0")
	fact := NewDummyBallotFact()

	fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)
	fact.H = nil

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "empty Fact")
}

func (t *testVoteproof) TestUnknownFactHash() {
	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact := NewDummyBallotFact()
	fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

	unknownFact := NewDummyBallotFact()
	unknownFactSign := t.signFact(n1.Address(), n1.Privatekey(), unknownFact, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs),
			NewBaseSignedBallotFact(unknownFact, unknownFactSign),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteproof) TestFactNotFound() {
	n0 := RandomNode("n0")

	fact := NewDummyBallotFact()

	newfact := NewDummyBallotFact()
	newfs := t.signFact(n0.Address(), n0.Privatekey(), newfact, nil)

	threshold, _ := NewThreshold(1, 40)
	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(newfact, newfs),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "missing fact found")
}

func (t *testVoteproof) TestSuplusFacts() {
	n0 := RandomNode("n0")

	fact := NewDummyBallotFact()
	fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact,
		facts:          []BallotFact{fact, fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "unknown facts found")
}

func (t *testVoteproof) TestNotYetButNot() {
	n0 := RandomNode("n0")

	fact := NewDummyBallotFact()
	fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), RandomStringAddress(), RandomStringAddress()},
		thresholdRatio: t.threshold.Ratio,
		result:         VoteResultDraw,
		majority:       fact,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result should be not-yet")
}

func (t *testVoteproof) TestDrawButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact0 := NewDummyBallotFact()
	fs0 := t.signFact(n0.Address(), n0.Privatekey(), fact0, nil)

	fact1 := NewDummyBallotFact()
	fs1 := t.signFact(n1.Address(), n1.Privatekey(), fact1, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), n1.Address()},
		thresholdRatio: threshold.Ratio,
		result:         VoteResultMajority,
		majority:       fact0,
		facts:          []BallotFact{fact0, fact1},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact0, fs0),
			NewBaseSignedBallotFact(fact1, fs1),
		},
		finishedAt: localtime.UTCNow(),
	}
	err := vp.IsValid(nil)
	t.Contains(err.Error(), "result mismatch")
	t.Contains(err.Error(), "DRAW")
}

func (t *testVoteproof) TestMajorityButNot() {
	threshold, _ := NewThreshold(2, 80)

	n0 := RandomNode("n0")
	n1 := RandomNode("n1")

	fact := NewDummyBallotFact()
	fs0 := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

	fs1 := t.signFact(n1.Address(), n1.Privatekey(), fact, nil)

	vp := VoteproofV0{
		stage:          StageINIT,
		suffrages:      []Address{n0.Address(), n1.Address()},
		thresholdRatio: threshold.Ratio,
		result:         VoteResultDraw,
		facts:          []BallotFact{fact},
		votes: []SignedBallotFact{
			NewBaseSignedBallotFact(fact, fs0),
			NewBaseSignedBallotFact(fact, fs1),
		},
		finishedAt: localtime.UTCNow(),
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

		fact := NewDummyBallotFact()
		fs := t.signFact(n0.Address(), n0.Privatekey(), fact, nil)

		return VoteproofV0{
			height:         height,
			round:          round,
			stage:          stage,
			thresholdRatio: t.threshold.Ratio,
			result:         VoteResultMajority,
			majority:       fact,
			facts:          []BallotFact{fact},
			votes: []SignedBallotFact{
				NewBaseSignedBallotFact(fact, fs),
			},
			finishedAt: localtime.UTCNow(),
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
