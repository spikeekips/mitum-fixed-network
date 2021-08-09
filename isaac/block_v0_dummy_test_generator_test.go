package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/stretchr/testify/suite"
)

type dummyOperationProcessor struct {
	pool            *storage.Statepool
	beforeProcessed func(state.Processor) error
	afterProcessed  func(state.Processor) error
}

func (opp dummyOperationProcessor) New(pool *storage.Statepool) prprocessor.OperationProcessor {
	return dummyOperationProcessor{
		pool:            pool,
		beforeProcessed: opp.beforeProcessed,
		afterProcessed:  opp.afterProcessed,
	}
}

func (opp dummyOperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	return op, nil
}

func (opp dummyOperationProcessor) Process(op state.Processor) error {
	if opp.beforeProcessed != nil {
		if err := opp.beforeProcessed(op); err != nil {
			return err
		}
	}

	if err := op.Process(opp.pool.Get, opp.pool.Set); err != nil {
		return err
	}

	if opp.afterProcessed == nil {
		return nil
	}

	return opp.afterProcessed(op)
}

func (opp dummyOperationProcessor) Close() error {
	return nil
}

func (opp dummyOperationProcessor) Cancel() error {
	return nil
}

type testBlockV0DummyGenerator struct {
	BaseTest
}

func (t *testBlockV0DummyGenerator) TestCreate() {
	all := t.Locals(3)

	for _, l := range all {
		t.NoError(l.Database().Clean())
	}

	var suffrage base.Suffrage
	{
		nodes := make([]base.Address, len(all))
		for i := range all {
			nodes[i] = all[i].Node().Address()
		}

		suffrage = t.Suffrage(all[0], all...)
	}

	lastHeight := base.Height(3)
	bg, err := NewDummyBlocksV0Generator(all[0], lastHeight, suffrage, all)
	t.NoError(err)

	t.NoError(bg.Generate(true))

	for i := int64(0); i < lastHeight.Int64(); i++ {
		hashes := map[string]struct{}{}
		for nodeid, l := range all {
			bs := l.BlockData().(*localfs.BlockData)
			_, blk, err := localfs.LoadBlock(bs, base.Height(i))
			t.NoError(err)

			t.NoError(err, "node=%d height=%d", nodeid, i)
			t.NotNil(blk, "node=%d height=%d", nodeid, i)
			t.NoError(blk.IsValid(all[0].Policy().NetworkID()), "height=%d", i)

			hashes[blk.Hash().String()] = struct{}{}
		}

		t.Equal(1, len(hashes), "check block hashes are matched")
	}
}

func (t *testBlockV0DummyGenerator) TestblockdataCleanByHeight() {
	local := t.Locals(1)[0]

	lastManifest, _, _ := local.Database().LastManifest()

	found, err := local.BlockData().Exists(lastManifest.Height())
	t.NoError(err)
	t.True(found)

	t.NoError(blockdata.CleanByHeight(local.Database(), local.BlockData(), lastManifest.Height()))

	l, _, _ := local.Database().LastManifest()
	t.Equal(lastManifest.Height()-1, l.Height())

	found, err = local.BlockData().Exists(lastManifest.Height())
	t.NoError(err)
	t.False(found)
}

func TestBlockV0DummyGenerator(t *testing.T) {
	suite.Run(t, new(testBlockV0DummyGenerator))
}
