package element

import (
	"encoding/json"
	"testing"

	"github.com/spikeekips/mitum/common"
	"github.com/stretchr/testify/suite"
)

type testTransaction struct {
	suite.Suite
}

func (t *testTransaction) TestNew() {
	checkpoint := []byte("showme")
	baseFee := common.NewBig(10)
	source := common.RandomSeed()

	tx := NewTransaction(source.Address(), checkpoint, baseFee, nil)

	t.Equal(source.Address(), tx.Source)
	t.Equal(common.NewBig(0), tx.Fee)
	t.Equal(checkpoint, tx.Checkpoint)
	t.NotEmpty(tx.CreatedAt)
}

func (t *testTransaction) TestJSON() {
	checkpoint := []byte("showme")
	baseFee := common.NewBig(10)
	source := common.RandomSeed()

	tx := NewTransaction(source.Address(), checkpoint, baseFee, nil)

	b, err := json.Marshal(tx)
	t.NoError(err)

	var ntx Transaction
	err = json.Unmarshal(b, &ntx)
	t.NoError(err)

	t.Equal(tx.Source, ntx.Source)
	t.Equal(tx.Fee, ntx.Fee)
	t.Equal(tx.Checkpoint, ntx.Checkpoint)
	t.Equal(tx.CreatedAt.String(), ntx.CreatedAt.String())
}

func TestTransaction(t *testing.T) {
	suite.Run(t, new(testTransaction))
}
