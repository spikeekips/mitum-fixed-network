package isaac

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testBlockData struct {
	BaseTest
}

func (t *testBlockData) processSession(local *Local, ss *localfs.Session) {
	var blk block.Block
	{
		i, err := block.NewTestBlockV0(ss.Height(), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
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
			err := tg.Add(tree.NewBaseFixedTreeNode(uint64(i), sts[i].Hash().Bytes()))
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
			base.RandomNode(util.UUID().String()),
			base.RandomNode(util.UUID().String()),
			base.RandomNode(util.UUID().String()),
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

		t.NoError(ss.SetProposal(pr))
	}
}

func (t *testBlockData) createFile(root, p string) *os.File {
	f, err := os.OpenFile(filepath.Join(root, p), os.O_CREATE|os.O_WRONLY, localfs.DefaultFilePermission)
	t.NoError(err)

	return f
}

func (t *testBlockData) newBlockData(root string) *localfs.BlockData {
	st := localfs.NewBlockData(root, t.JSONEnc)
	t.NoError(st.Initialize())

	return st
}

func (t *testBlockData) TestSaveSession() {
	local := t.Locals(1)[0]

	st := t.newBlockData(t.Root)

	ss, err := st.NewSession(33)
	t.NoError(err)

	t.processSession(local, ss.(*localfs.Session))

	_, err = st.SaveSession(ss)
	t.NoError(err)
}

func (t *testBlockData) TestHeightDirectoryAlreadyExists() {
	local := t.Locals(1)[0]

	st := t.newBlockData(t.Root)

	ss, err := st.NewSession(33)
	t.NoError(err)

	t.processSession(local, ss.(*localfs.Session))

	// NOTE create height directory and create new file
	var insidefile string
	{
		target := filepath.Join(t.Root, st.HeightDirectory(33, false))
		t.NoError(st.CreateDirectory(target))

		insidefile = filepath.Join(target, "a.map")
		f := t.createFile("", insidefile)
		t.NoError(f.Close())

		_, err = os.Stat(insidefile)
		t.NoError(err)
	}

	_, err = st.SaveSession(ss)
	t.NoError(err)

	_, err = os.Stat(insidefile)
	t.True(os.IsNotExist(err))
}

func (t *testBlockData) TestClean() {
	st := t.newBlockData(t.Root)

	touch := func(a string) {
		f := t.createFile(t.Root, a)
		f.Close()
	}

	exists := func(p string) bool {
		_, err := os.Stat(p)
		if err == nil {
			return true
		}

		if os.IsNotExist(err) {
			return false
		}

		panic(err)
	}

	touch("a")
	touch("b")
	touch("c")

	t.NoError(st.Clean(false))

	t.True(exists(t.Root))
	t.False(exists("a"))
	t.False(exists("b"))
	t.False(exists("c"))

	t.NoError(st.Clean(true))
	t.False(exists(t.Root))
}

func (t *testBlockData) TestRemove() {
	local := t.Locals(1)[0]

	st := t.newBlockData(t.Root)

	for i := int64(33); i < 36; i++ {
		ss, err := st.NewSession(base.Height(i))
		t.NoError(err)

		t.processSession(local, ss.(*localfs.Session))

		_, err = st.SaveSession(ss)
		t.NoError(err)
	}

	for i := int64(33); i < 36; i++ {
		found, err := st.Exists(base.Height(i))
		t.NoError(err)
		t.True(found)
	}

	err := st.Remove(22)
	t.Error(err)
	t.True(xerrors.Is(err, util.NotFoundError))

	t.NoError(st.Remove(34))
	found, err := st.Exists(34)
	t.NoError(err)
	t.False(found)

	target := st.HeightDirectory(34, true)

	files, err := os.ReadDir(target)
	t.NoError(err)
	t.NotEmpty(files)
}

func (t *testBlockData) TestRemoveAll() {
	local := t.Locals(1)[0]

	st := t.newBlockData(t.Root)

	for i := int64(33); i < 36; i++ {
		ss, err := st.NewSession(base.Height(i))
		t.NoError(err)

		t.processSession(local, ss.(*localfs.Session))

		_, err = st.SaveSession(ss)
		t.NoError(err)
	}

	for i := int64(33); i < 36; i++ {
		found, err := st.Exists(base.Height(i))
		t.NoError(err)
		t.True(found)
	}

	err := st.RemoveAll(22)
	t.Error(err)
	t.True(xerrors.Is(err, util.NotFoundError))

	t.NoError(st.RemoveAll(34))

	found, err := st.Exists(34)
	t.NoError(err)
	t.False(found)

	target := st.HeightDirectory(34, true)
	_, err = os.Stat(target)
	t.True(os.IsNotExist(err))
}

func TestBlockData(t *testing.T) {
	suite.Run(t, new(testBlockData))
}
