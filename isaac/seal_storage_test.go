package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/node"
	"github.com/stretchr/testify/suite"
)

type testSealStorage struct {
	suite.Suite
}

func (t *testSealStorage) newBallot() Ballot {
	home := node.NewRandomHome()

	ballot, _ := NewINITBallot(
		home.Address(),
		NewRandomBlockHash(),
		Round(0),
		NewBlockHeight(1),
		NewRandomBlockHash(),
		Round(1),
		NewRandomProposalHash(),
	)

	_ = ballot.Sign(home.PrivateKey(), nil)

	return ballot
}

func (t *testSealStorage) TestSave() {
	st := NewTSealStorage()

	ballot := t.newBallot()
	err := st.Save(ballot)
	t.NoError(err)

	t.True(st.Has(ballot.Hash()))
	loaded, found := st.Get(ballot.Hash())

	t.True(found)
	t.True(ballot.Hash().Equal(loaded.Hash()))
	t.True(ballot.Equal(loaded))
}

func (t *testSealStorage) TestHas() {
	st := NewTSealStorage()

	ballot := t.newBallot()
	err := st.Save(ballot)
	t.NoError(err)

	t.True(st.Has(ballot.Hash()))

	t.False(st.Has(NewRandomBallotHash()))
	_, found := st.Get(NewRandomBallotHash())
	t.False(found)
}

func (t *testSealStorage) TestNilSeal() {
	st := NewTSealStorage()

	err := st.Save(nil)
	t.Contains(err.Error(), "nil")
}

func TestSealStorage(t *testing.T) {
	suite.Run(t, new(testSealStorage))
}
