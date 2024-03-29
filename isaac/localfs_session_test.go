package isaac

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testBlockdataLocalFSSession struct {
	BaseTest
}

func (t *testBlockdataLocalFSSession) openFile(p string) (io.ReadCloser, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	return gzip.NewReader(f)
}

func (t *testBlockdataLocalFSSession) checkSessionFile(ss *localfs.Session, dataType string) string {
	item, found := ss.MapData().Item(dataType)
	t.True(found, dataType)
	t.NoError(item.IsValid(nil), dataType)

	p := item.URL()[7:]
	fi, err := os.Stat(p)
	t.NoError(err)
	t.Equal(localfs.DefaultFilePermission, fi.Mode())
	t.True(fi.Size() > 0)
	t.Equal(filepath.Base(p), fi.Name())

	return p
}

func (t *testBlockdataLocalFSSession) TestAddOperationsButNotFinished() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	local := t.Locals(1)[0]
	ops := t.NewOperations(local, 1)
	t.NoError(ss.AddOperations(ops...))

	item, found := ss.MapData().Item("operations")
	t.True(found)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "empty data type of map item")
}

func (t *testBlockdataLocalFSSession) TestAddOperationsFinishedWithClose() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	local := t.Locals(1)[0]
	ops := t.NewOperations(local, 10)

	t.NoError(ss.AddOperations(ops...))
	t.NoError(ss.CloseOperations())

	p := t.checkSessionFile(ss, block.BlockdataOperations)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	uops, err := writer.ReadOperations(r)
	t.NoError(err)
	t.Equal(len(ops), len(uops))

	for i := range ops {
		a := ops[i]
		b := uops[i]

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.Hint().Equal(b.Hint()))
		t.True(a.LastSignedAt().Equal(b.LastSignedAt()))
		t.Equal(a.Fact(), b.Fact())
		for j := range a.Signs() {
			as := a.Signs()[j]
			bs := b.Signs()[j]
			t.Equal(as.Bytes(), bs.Bytes())
		}
	}
}

func (t *testBlockdataLocalFSSession) TestAddStatesButNotFinished() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	sts := make([]state.State, 2)
	for i := 0; i < 2; i++ {
		sts[i] = t.NewState(33)
	}

	t.NoError(ss.AddStates(sts...))

	item, found := ss.MapData().Item("states")
	t.True(found)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "empty data type of map item")
}

func (t *testBlockdataLocalFSSession) TestAddStatesFinishedWithClose() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	sts := make([]state.State, 5)
	for i := 0; i < 5; i++ {
		sts[i] = t.NewState(33)
	}

	t.NoError(ss.AddStates(sts...))
	t.NoError(ss.CloseStates())

	p := t.checkSessionFile(ss, block.BlockdataStates)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	usts, err := writer.ReadStates(r)
	t.NoError(err)
	t.Equal(5, len(usts))

	for i := range sts {
		a := sts[i]
		b := usts[i]

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.Hint().Equal(b.Hint()))
		t.Equal(a.Key(), b.Key())
		t.Equal(a.Height(), b.Height())
		t.True(a.Value().Equal(b.Value()))
	}
}

func (t *testBlockdataLocalFSSession) TestSetStatesTree() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	sts := make([]state.State, 5)
	for i := 0; i < 5; i++ {
		sts[i] = t.NewState(33)
	}

	tg := tree.NewFixedTreeGenerator(uint64(len(sts)))
	for i := range sts {
		err := tg.Add(state.NewFixedTreeNode(uint64(i), sts[i].Hash().Bytes()))
		t.NoError(err)
	}

	tr, err := tg.Tree()
	t.NoError(err)
	t.NoError(tr.IsValid(nil))

	t.NoError(ss.SetStatesTree(tr))

	p := t.checkSessionFile(ss, block.BlockdataStatesTree)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	utr, err := writer.ReadStatesTree(r)
	t.NoError(err)

	t.NoError(utr.IsValid(nil))
	t.Equal(tr.Len(), utr.Len())
	t.True(tr.Hint().Equal(utr.Hint()))

	t.NoError(tr.Traverse(func(no tree.FixedTreeNode) (bool, error) {
		if i, err := utr.Node(no.Index()); err != nil {
			return false, err
		} else if !no.Equal(i) {
			return false, errors.Errorf("different node found")
		}

		return true, nil
	}))
}

func (t *testBlockdataLocalFSSession) TestSetINITVoteproof() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	local := t.Locals(1)[0]

	ib := t.NewINITBallot(local, base.Round(0), nil)
	vp, err := t.NewVoteproof(base.StageINIT, ib.Fact(), local)
	t.NoError(err)

	t.NoError(ss.SetINITVoteproof(vp))

	p := t.checkSessionFile(ss, block.BlockdataINITVoteproof)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	uvp, err := writer.ReadINITVoteproof(r)
	t.NoError(err)

	t.CompareVoteproof(vp, uvp)
}

func (t *testBlockdataLocalFSSession) TestSetACCEPTVoteproof() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	local := t.Locals(1)[0]

	ab := t.NewACCEPTBallot(local, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256(), nil)
	vp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), local)
	t.NoError(err)

	t.NoError(ss.SetACCEPTVoteproof(vp))

	p := t.checkSessionFile(ss, block.BlockdataACCEPTVoteproof)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	uvp, err := writer.ReadACCEPTVoteproof(r)
	t.NoError(err)

	t.CompareVoteproof(vp, uvp)
}

func (t *testBlockdataLocalFSSession) TestSetProposal() {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	local := t.Locals(1)[0]

	pr := t.NewProposal(local, base.Round(0), []valuehash.Hash{
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	}, nil)

	t.NoError(ss.SetProposal(pr.SignedFact()))

	p := t.checkSessionFile(ss, block.BlockdataProposal)
	writer := blockdata.NewDefaultWriter(t.JSONEnc)
	r, err := t.openFile(p)
	t.NoError(err)

	upr, err := writer.ReadProposal(r)
	t.NoError(err)

	t.CompareProposalFact(pr.Fact(), upr.Fact().(base.ProposalFact))
}

func (t *testBlockdataLocalFSSession) saveBlock(local *Local) (*localfs.Session, block.Block) {
	ss, err := localfs.NewSession(t.Root, blockdata.NewDefaultWriter(t.JSONEnc), base.Height(33))
	t.NoError(err)

	var blk block.Block
	{
		i, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
		t.NoError(err)
		blk = i

		t.NoError(ss.SetManifest(blk.Manifest()))
	}

	{
		ops := t.NewOperations(local, 3)

		t.NoError(ss.AddOperations(ops...))
		t.NoError(ss.CloseOperations())

		tg := tree.NewFixedTreeGenerator(uint64(len(ops)))
		for i := range ops {
			err := tg.Add(operation.NewFixedTreeNode(uint64(i), ops[i].Hash().Bytes(), true, nil))
			t.NoError(err)
		}

		tr, err := tg.Tree()
		t.NoError(err)
		t.NoError(tr.IsValid(nil))

		t.NoError(ss.SetOperationsTree(tr))
	}

	{
		sts := make([]state.State, 5)
		for i := 0; i < 5; i++ {
			sts[i] = t.NewState(33)
		}

		t.NoError(ss.AddStates(sts...))
		t.NoError(ss.CloseStates())

		tg := tree.NewFixedTreeGenerator(uint64(len(sts)))
		for i := range sts {
			err := tg.Add(state.NewFixedTreeNode(uint64(i), sts[i].Hash().Bytes()))
			t.NoError(err)
		}

		tr, err := tg.Tree()
		t.NoError(err)
		t.NoError(tr.IsValid(nil))

		t.NoError(ss.SetStatesTree(tr))
	}

	{
		ib := t.NewINITBallot(local, base.Round(0), nil)
		vp, err := t.NewVoteproof(base.StageINIT, ib.Fact(), local)
		t.NoError(err)

		t.NoError(ss.SetINITVoteproof(vp))
	}

	{
		ab := t.NewACCEPTBallot(local, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256(), nil)
		vp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), local)
		t.NoError(err)

		t.NoError(ss.SetACCEPTVoteproof(vp))
	}

	{
		nodes := []base.Node{
			node.RandomNode(util.UUID().String()),
			node.RandomNode(util.UUID().String()),
			node.RandomNode(util.UUID().String()),
		}
		sf := block.NewSuffrageInfoV0(nodes[0].Address(), nodes)

		t.NoError(ss.SetSuffrageInfo(sf))
	}

	{
		pr := t.NewProposal(local, base.Round(0), []valuehash.Hash{
			valuehash.RandomSHA256(),
			valuehash.RandomSHA256(),
			valuehash.RandomSHA256(),
			valuehash.RandomSHA256(),
		}, nil)

		t.NoError(ss.SetProposal(pr.SignedFact()))
	}

	return ss, blk
}

func (t *testBlockdataLocalFSSession) TestDone() {
	local := t.Locals(1)[0]
	ss, blk := t.saveBlock(local)

	defer ss.Cancel()

	mapData, err := ss.Done()
	t.NoError(err)
	_, err = os.Stat(t.Root)
	t.NoError(err)

	t.NoError(mapData.IsValid(nil))
	t.NoError(mapData.Exists("/"))

	t.True(mapData.Block().Equal(blk.Hash()))
}

func (t *testBlockdataLocalFSSession) TestImport() {
	local := t.Locals(1)[0]
	ss, blk := t.saveBlock(local)

	defer ss.Cancel()

	mapData, err := ss.Done()
	t.NoError(err)

	t.NoError(mapData.IsValid(nil))
	t.NoError(mapData.Exists("/"))
	t.True(mapData.Block().Equal(blk.Hash()))

	newroot, err := os.MkdirTemp("", "localfs-")
	t.NoError(err)
	defer os.RemoveAll(newroot)

	nss, err := localfs.NewSession(newroot, blockdata.NewDefaultWriter(t.JSONEnc), mapData.Height())
	t.NoError(err)

	for i := range block.Blockdata {
		dataType := block.Blockdata[i]

		p := t.checkSessionFile(ss, dataType)
		r, err := t.openFile(p)
		t.NoError(err)

		_, err = nss.Import(dataType, r)
		t.NoError(err)

		_ = t.checkSessionFile(nss, dataType)
	}

	newMapData := nss.MapData()
	_, err = newMapData.UpdateHash()
	t.Contains(err.Error(), "nil can not be checked")

	err = newMapData.IsValid(nil)
	t.Contains(err.Error(), "nil can not be checked")

	newMapData = newMapData.SetBlock(mapData.Block())
	newMapData, err = newMapData.UpdateHash()
	t.NoError(err)
	t.NoError(newMapData.IsValid(nil))

	for i := range block.Blockdata {
		dataType := block.Blockdata[i]

		a, found := ss.MapData().Item(dataType)
		t.True(found)
		b, found := nss.MapData().Item(dataType)
		t.True(found)

		t.Equal(a.Type(), b.Type())
		t.Equal(a.Checksum(), b.Checksum())
		t.NotEqual(a.URL(), b.URL())
	}
}

func TestBlockdataLocalFSSession(t *testing.T) {
	suite.Run(t, new(testBlockdataLocalFSSession))
}
