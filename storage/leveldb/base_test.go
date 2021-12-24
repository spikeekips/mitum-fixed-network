package leveldbstorage

import (
	"context"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testDatabase struct {
	storage.BaseTestDatabase
	database *Database
}

func (t *testDatabase) SetupTest() {
	t.database = NewMemDatabase(t.Encs, t.JSONEnc)
}

func (t *testDatabase) TestNew() {
	t.Implements((*storage.Database)(nil), t.database)
}

func (t *testDatabase) saveBlockdataMap(st *Database, bd block.BlockdataMap) error {
	if b, err := marshal(bd, st.enc); err != nil {
		return err
	} else {
		return st.db.Put(leveldbBlockdataMapKey(bd.Height()), b, nil)
	}
}

func (t *testDatabase) TestLastBlock() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))

	bd := t.NewBlockdataMap(blk.Height(), blk.Hash(), true)
	t.NoError(bs.Commit(context.Background(), bd))

	loaded, found, err := t.database.lastBlock()
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)

	ubd, found, err := t.database.BlockdataMap(blk.Height())
	t.NoError(err)
	t.True(found)

	block.CompareBlockdataMap(t.Assert(), bd, ubd)
}

func (t *testDatabase) TestLastManifest() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockdataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(blk.Manifest(), loaded)
}

func (t *testDatabase) TestLoadBlockByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	{
		b, err := t.JSONEnc.Marshal(blk)
		t.NoError(err)

		hb := encodeWithEncoder(b, t.JSONEnc)

		key := leveldbBlockHashKey(blk.Hash())
		t.NoError(t.database.db.Put(key, hb, nil))
		t.NoError(t.database.db.Put(leveldbBlockHeightKey(blk.Height()), key, nil))
	}

	loaded, found, err := t.database.block(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testDatabase) TestLoadManifestByHash() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockdataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestLoadManifestByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockdataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.ManifestByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.Implements((*block.Manifest)(nil), loaded)
	_, isBlock := loaded.(block.Block)
	t.False(isBlock)

	t.CompareManifest(blk, loaded)
}

func (t *testDatabase) TestLoadBlockByHeight() {
	// store first
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	bs, err := t.database.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.SetBlock(context.Background(), blk))
	t.NoError(bs.Commit(context.Background(), t.NewBlockdataMap(blk.Height(), blk.Hash(), true)))

	loaded, found, err := t.database.blockByHeight(blk.Height())
	t.NoError(err)
	t.True(found)

	t.CompareBlock(blk, loaded)
}

func (t *testDatabase) TestNewOperationSeals() {
	var ops []operation.Operation
	var seals []operation.Seal
	for i := 0; i < 10; i++ {
		opsl := t.newOperationSeal()
		seals = append(seals, opsl)

		l := opsl.Operations()
		for j := range l {
			ops = append(ops, l[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	var collected []operation.Operation
	t.NoError(t.database.StagedOperations(
		func(op operation.Operation) (bool, error) {
			collected = append(collected, op)

			t.NoError(op.IsValid(nil))

			return true, nil
		},
		true,
	))

	t.Equal(len(ops), len(collected))

	for i := range collected {
		a := ops[i]
		b := collected[i]

		found, err := t.database.HasStagedOperation(b.Fact().Hash())
		t.NoError(err)
		t.True(found)

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.Fact().Hash().Equal(b.Fact().Hash()))
	}
}

func (t *testDatabase) TestStagedOperationsByFact() {
	var ops []operation.Operation
	var seals []operation.Seal
	for i := 0; i < 10; i++ {
		opsl := t.newOperationSeal()
		seals = append(seals, opsl)

		l := opsl.Operations()
		for j := range l {
			ops = append(ops, l[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	var facts []valuehash.Hash
	for i := range ops[:3] {
		facts = append(facts, ops[i].Fact().Hash())
	}

	rops, err := t.database.StagedOperationsByFact(facts)
	t.NoError(err)

	t.Equal(3, len(rops))

	for i := range rops {
		op := rops[i]

		t.True(facts[i].Equal(op.Fact().Hash()))
	}

	l, err := t.database.StagedOperationsByFact([]valuehash.Hash{valuehash.RandomSHA256()})
	t.NoError(err)
	t.Equal(0, len(l))
}

func (t *testDatabase) TestUnstagedOperations() {
	var ops []operation.Operation
	var seals []operation.Seal
	for i := 0; i < 10; i++ {
		opsl := t.newOperationSeal()
		seals = append(seals, opsl)

		l := opsl.Operations()
		for j := range l {
			ops = append(ops, l[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	var inserted int
	_ = t.database.iter(
		nil,
		func(key, _ []byte) (bool, error) {
			inserted++
			return true, nil
		},
		false,
	)

	var facts []valuehash.Hash
	for i := range ops[:3] {
		facts = append(facts, ops[i].Fact().Hash())
	}

	t.NoError(t.database.UnstagedOperations(facts))

	for i := range facts {
		found, err := t.database.HasStagedOperation(facts[i])
		t.NoError(err)
		t.False(found)
	}

	l, err := t.database.StagedOperationsByFact(facts)
	t.NoError(err)
	t.Equal(0, len(l))

	var lefts int
	_ = t.database.iter(
		nil,
		func(key, _ []byte) (bool, error) {
			lefts++
			return true, nil
		},
		false,
	)

	t.Equal(inserted-6, lefts)
}

func (t *testDatabase) TestStagedOperationsLimit() {
	var ops []operation.Operation
	var seals []operation.Seal
	for i := 0; i < 10; i++ {
		opsl := t.newOperationSeal()
		seals = append(seals, opsl)

		l := opsl.Operations()
		for j := range l {
			ops = append(ops, l[j])
		}
	}
	t.NoError(t.database.NewOperationSeals(seals))

	var collected []valuehash.Hash
	t.NoError(t.database.StagedOperations(
		func(op operation.Operation) (bool, error) {
			if len(collected) == 3 {
				return false, nil
			}
			collected = append(collected, op.Fact().Hash())

			return true, nil
		},
		true,
	))

	t.Equal(3, len(collected))

	for i := range collected {
		a := ops[i].Fact().Hash()
		b := collected[i]

		t.True(a.Equal(b))
	}
}

func (t *testDatabase) newOperationSeal() operation.Seal {
	token := []byte("this-is-token")
	op, err := operation.NewKVOperation(t.PK, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewBaseSeal(t.PK, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *testDatabase) TestHasOperation() {
	fact := valuehash.RandomSHA256()

	{ // store
		raw, err := t.database.enc.Marshal(fact)
		t.NoError(err)
		t.database.db.Put(
			leveldbOperationFactHashKey(fact),
			encodeWithEncoder(raw, t.database.enc),
			nil,
		)
	}

	{
		found, err := t.database.HasOperationFact(fact)
		t.NoError(err)
		t.True(found)
	}

	{ // unknown
		found, err := t.database.HasOperationFact(valuehash.RandomSHA256())
		t.NoError(err)
		t.False(found)
	}
}

func (t *testDatabase) TestInfo() {
	key := util.UUID().String()
	b := util.UUID().Bytes()

	_, found, err := t.database.Info(key)
	t.NoError(err)
	t.False(found)

	t.NoError(t.database.SetInfo(key, b))

	ub, found, err := t.database.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(b, ub)

	nb := util.UUID().Bytes()
	t.NoError(t.database.SetInfo(key, nb))

	unb, found, err := t.database.Info(key)
	t.NoError(err)
	t.True(found)
	t.Equal(nb, unb)
}

func (t *testDatabase) TestLocalBlockdataMapsByHeight() {
	isLocal := func(height base.Height) bool {
		switch height {
		case 33, 36, 39:
			return true
		default:
			return false
		}
	}

	for i := base.Height(33); i < 40; i++ {
		bd := t.NewBlockdataMap(i, valuehash.RandomSHA256(), isLocal(i))
		t.NoError(t.saveBlockdataMap(t.database, bd))
	}

	for i := base.Height(33); i < 40; i++ {
		bd, found, err := t.database.BlockdataMap(i)
		t.NoError(err)
		t.True(found)
		t.NotNil(bd)
	}

	err := t.database.LocalBlockdataMapsByHeight(36, func(bd block.BlockdataMap) (bool, error) {
		t.True(bd.Height() > 35)
		t.True(isLocal(bd.Height()))
		t.True(bd.IsLocal())

		return true, nil
	})
	t.NoError(err)
}

func TestLeveldbDatabase(t *testing.T) {
	suite.Run(t, new(testDatabase))
}
