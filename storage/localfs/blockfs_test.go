package localfs

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testBlock struct {
	suite.Suite
	BaseTestLocalFS
	Encs    *encoder.Encoders
	JSONEnc encoder.Encoder
	BSONEnc encoder.Encoder
}

func (t *testBlock) SetupTest() {
	t.Encs = encoder.NewEncoders()
	t.JSONEnc = jsonenc.NewEncoder()
	t.BSONEnc = bsonenc.NewEncoder()

	t.NoError(t.Encs.AddEncoder(t.JSONEnc))
	t.NoError(t.Encs.AddEncoder(t.BSONEnc))

	_ = t.Encs.AddHinter(base.StringAddress(""))
	_ = t.Encs.AddHinter(base.BaseNodeV0{})
	_ = t.Encs.AddHinter(block.SuffrageInfoV0{})
	_ = t.Encs.AddHinter(key.BTCPublickeyHinter)
	_ = t.Encs.AddHinter(block.BlockV0{})
	_ = t.Encs.AddHinter(block.ManifestV0{})
	_ = t.Encs.AddHinter(block.ConsensusInfoV0{})
	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(base.VoteproofV0{})
	_ = t.Encs.AddHinter(seal.DummySeal{})
	_ = t.Encs.AddHinter(operation.BaseSeal{})
	_ = t.Encs.AddHinter(operation.BaseFactSign{})
	_ = t.Encs.AddHinter(operation.KVOperation{})
	_ = t.Encs.AddHinter(operation.KVOperationFact{})
	_ = t.Encs.AddHinter(tree.FixedTree{})
}

func (t *testBlock) TestFileHash() {
	sh := sha256.New()
	_, err := sh.Write([]byte("showme\n"))
	t.NoError(err)

	b := sh.Sum(nil)
	t.Equal("6da6a88572492a88254b53cf6f504be29207b42b8523330d3e3ab4125b4c71c4", fmt.Sprintf("%x", b))

	// NOTE equivalant with `echo "showme" | shasum -a 256 -b`
}

func (t *testBlock) TestNew() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.AddManifest(blk.Height(), blk.Hash(), blk.Manifest()))
	t.NoError(bs.AddOperationsTree(blk.Height(), blk.Hash(), blk.OperationsTree()))
	t.NoError(bs.AddOperations(blk.Height(), blk.Hash(), blk.Operations()))
	t.NoError(bs.AddStatesTree(blk.Height(), blk.Hash(), blk.StatesTree()))
	t.NoError(bs.AddStates(blk.Height(), blk.Hash(), blk.States()))
	t.NoError(bs.AddINITVoteproof(blk.Height(), blk.Hash(), blk.ConsensusInfo().INITVoteproof()))
	t.NoError(bs.AddACCEPTVoteproof(blk.Height(), blk.Hash(), blk.ConsensusInfo().ACCEPTVoteproof()))
	t.NoError(bs.AddSuffrage(blk.Height(), blk.Hash(), blk.ConsensusInfo().SuffrageInfo()))
	t.NoError(bs.AddProposal(blk.Height(), blk.Hash(), blk.ConsensusInfo().Proposal()))

	t.NoError(bs.Commit(blk.Height(), blk.Hash()))

	h, err := bs.Exists(blk.Height())
	t.NoError(err)
	t.True(blk.Hash().Equal(h))

	t.NoError(bs.Remove(blk.Height()))
	_, err = bs.Exists(blk.Height())
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testBlock) TestPregenesisHeight() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.PreGenesisHeight, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.Add(blk))

	t.NoError(bs.Commit(blk.Height(), blk.Hash()))

	h, err := bs.Exists(blk.Height())
	t.NoError(err)
	t.True(blk.Hash().Equal(h))

	t.NoError(bs.Remove(blk.Height()))
	_, err = bs.Exists(blk.Height())
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testBlock) TestNewAndCancel() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.Add(blk))

	_, err = bs.Exists(blk.Height())
	t.True(xerrors.Is(err, storage.NotFoundError))

	var foundInUnstaged bool
	fs.Walk("/tmp", func(fp string, fi os.FileInfo) error {
		foundInUnstaged = true
		return nil
	})
	t.True(foundInUnstaged)

	t.NoError(bs.Cancel(blk.Height(), blk.Hash()))

	foundInUnstaged = false
	fs.Walk("/tmp", func(fp string, fi os.FileInfo) error {
		foundInUnstaged = true
		return nil
	})
	t.False(foundInUnstaged)
}

func (t *testBlock) TestAdd() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.Add(blk))

	// will override
	nblk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.AddManifest(blk.Height(), blk.Hash(), nblk.Manifest()))

	t.NoError(bs.Commit(blk.Height(), blk.Hash()))

	manifest, err := bs.LoadManifest(blk.Height())
	t.NoError(err)

	t.Equal(blk.Manifest().Height(), manifest.Height())
	t.True(nblk.Manifest().Hash().Equal(manifest.Hash()))
}

func (t *testBlock) TestLoad() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.Add(blk))

	t.NoError(bs.Commit(blk.Height(), blk.Hash()))

	manifest, err := bs.LoadManifest(blk.Height())
	t.NoError(err)

	t.True(blk.Manifest().Hash().Equal(manifest.Hash()))

	ublk, err := bs.Load(blk.Height())
	t.NoError(err)

	t.Equal(blk.Height(), ublk.Height())
	t.True(blk.Hash().Equal(ublk.Hash()))
}

func (t *testBlock) TestOpen() {
	fs := t.FS()
	bs := storage.NewBlockFS(fs, t.JSONEnc.(*jsonenc.Encoder))

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(bs.Add(blk))

	t.NoError(bs.Commit(blk.Height(), blk.Hash()))

	r, isCompressed, err := bs.OpenManifest(blk.Height())
	t.NoError(err)
	t.True(isCompressed)

	gr, err := gzip.NewReader(r)
	t.NoError(err)

	b, err := ioutil.ReadAll(gr)
	t.NoError(err)
	t.NotNil(b)
}

func TestBlock(t *testing.T) {
	suite.Run(t, new(testBlock))
}
