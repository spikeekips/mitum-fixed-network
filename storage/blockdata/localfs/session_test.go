package localfs

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testSession struct {
	suite.Suite
	JSONEnc  *jsonenc.Encoder
	baseRoot string
	root     string
}

func (t *testSession) SetupSuite() {
	encs := encoder.NewEncoders()
	t.JSONEnc = jsonenc.NewEncoder()
	_ = encs.AddEncoder(t.JSONEnc)

	_ = encs.TestAddHinter(key.BTCPrivatekeyHinter)
	_ = encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = encs.TestAddHinter(base.StringAddress(""))
	_ = encs.TestAddHinter(ballot.INITV0{})
	_ = encs.TestAddHinter(ballot.INITFactV0{})
	_ = encs.TestAddHinter(ballot.ProposalV0{})
	_ = encs.TestAddHinter(ballot.ProposalFactV0{})
	_ = encs.TestAddHinter(ballot.SIGNV0{})
	_ = encs.TestAddHinter(ballot.SIGNFactV0{})
	_ = encs.TestAddHinter(ballot.ACCEPTV0{})
	_ = encs.TestAddHinter(ballot.ACCEPTFactV0{})
	_ = encs.TestAddHinter(base.VoteproofV0{})
	_ = encs.TestAddHinter(base.BaseVoteproofNodeFact{})
	_ = encs.TestAddHinter(node.BaseV0{})
	_ = encs.TestAddHinter(block.BlockV0{})
	_ = encs.TestAddHinter(block.ManifestV0{})
	_ = encs.TestAddHinter(block.ConsensusInfoV0{})
	_ = encs.TestAddHinter(block.SuffrageInfoV0{})
	_ = encs.TestAddHinter(operation.BaseFactSign{})
	_ = encs.TestAddHinter(operation.SealHinter)
	_ = encs.TestAddHinter(operation.KVOperationFact{})
	_ = encs.TestAddHinter(operation.KVOperation{})
	_ = encs.TestAddHinter(tree.FixedTree{})
	_ = encs.TestAddHinter(state.StateV0{})
	_ = encs.TestAddHinter(state.BytesValue{})
	_ = encs.TestAddHinter(state.DurationValue{})
	_ = encs.TestAddHinter(state.HintedValue{})
	_ = encs.TestAddHinter(state.NumberValue{})
	_ = encs.TestAddHinter(state.SliceValue{})
	_ = encs.TestAddHinter(state.StringValue{})

	p, err := os.MkdirTemp("", "localfs-")
	if err != nil {
		panic(err)
	}

	t.baseRoot = p
}

func (t *testSession) SetupTest() {
	p, err := os.MkdirTemp(t.baseRoot, "localfs-")
	if err != nil {
		panic(err)
	}

	t.root = p
}

func (t *testSession) TearDownSuite() {
	_ = os.RemoveAll(t.baseRoot)
}

func (t *testSession) loadFile(p string) ([]interface{}, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	gf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	var hinters []interface{}
	bd := bufio.NewReader(gf)
	for {
		l, err := bd.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
		}
		if len(l) > 0 {
			if i, err := t.JSONEnc.Decode(l); err != nil {
				return nil, err
			} else {
				hinters = append(hinters, i)
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return hinters, nil
}

func (t *testSession) checkSessionFile(ss *Session, dataType string) string {
	item, found := ss.mapData.Item(dataType)
	t.True(found, dataType)
	t.NoError(item.IsValid(nil), dataType)

	p := item.URL()[6:]
	fi, err := os.Stat(p)
	t.NoError(err)
	t.Equal(DefaultFilePermission, fi.Mode())
	t.True(fi.Size() > 0)
	t.Equal(filepath.Base(p), fi.Name())

	return p
}

func (t *testSession) TestRootNotExist() {
	_, err := NewSession(valuehash.RandomSHA256().String(), blockdata.NewDefaultWriter(t.JSONEnc), 10)
	t.Error(err)
	t.True(errors.Is(err, util.NotFoundError))
}

func (t *testSession) TestNew() {
	ss, err := NewSession(t.root, blockdata.NewDefaultWriter(t.JSONEnc), 10)
	t.NoError(err)
	defer ss.Cancel()

	t.Implements((*blockdata.Session)(nil), ss)

	t.Equal(base.Height(10), ss.Height())
}

func (t *testSession) TestSetManifest() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ss, err := NewSession(t.root, blockdata.NewDefaultWriter(t.JSONEnc), blk.Height())
	t.NoError(err)
	defer ss.Cancel()

	t.NoError(ss.SetManifest(blk.Manifest()))

	p := t.checkSessionFile(ss, "manifest")

	hinters, err := t.loadFile(p)
	t.NoError(err)
	t.Equal(1, len(hinters))
	t.Implements((*block.Manifest)(nil), hinters[0])

	loaded := hinters[0].(block.Manifest)

	t.Equal(blk.Height(), loaded.Height())
	t.Equal(blk.Round(), loaded.Round())
	t.True(blk.Hash().Equal(loaded.Hash()))
	t.True(blk.PreviousBlock().Equal(loaded.PreviousBlock()))
	t.True(blk.Proposal().Equal(loaded.Proposal()))
	t.True(blk.OperationsHash().Equal(loaded.OperationsHash()))
	t.True(blk.StatesHash().Equal(loaded.StatesHash()))
	t.True(blk.ConfirmedAt().Equal(loaded.ConfirmedAt()))
	t.True(blk.CreatedAt().Equal(loaded.CreatedAt()))
}

func (t *testSession) TestSetSuffrageInfo() {
	ss, err := NewSession(t.root, blockdata.NewDefaultWriter(t.JSONEnc), 33)
	t.NoError(err)
	defer ss.Cancel()

	nodes := []base.Node{
		node.RandomNode(util.UUID().String()),
		node.RandomNode(util.UUID().String()),
		node.RandomNode(util.UUID().String()),
	}
	sf := block.NewSuffrageInfoV0(nodes[0].Address(), nodes)

	t.NoError(ss.SetSuffrageInfo(sf))

	p := t.checkSessionFile(ss, "suffrage_info")

	hinters, err := t.loadFile(p)
	t.NoError(err)
	t.Equal(1, len(hinters))

	t.Implements((*block.SuffrageInfo)(nil), hinters[0])

	nsf := hinters[0].(block.SuffrageInfo)
	t.True(sf.Hint().Equal(nsf.Hint()))
	t.True(sf.Proposer().Equal(nsf.Proposer()))
	t.Equal(len(sf.Nodes()), len(nsf.Nodes()))
	for i := range sf.Nodes() {
		a := sf.Nodes()[i]
		b := nsf.Nodes()[i]

		t.True(a.Hint().Equal(b.Hint()))
		t.Equal(a.String(), b.String())
		t.True(a.Address().Equal(b.Address()))
		t.True(a.Publickey().Equal(b.Publickey()))
	}
}

func (t *testSession) TestDoneError() {
	ss, err := NewSession(t.root, blockdata.NewDefaultWriter(t.JSONEnc), 10)
	t.NoError(err)
	defer ss.Cancel()

	t.Equal(base.Height(10), ss.Height())
	_, err = ss.done()
	t.Contains(err.Error(), "nil can not be checked")
}

func (t *testSession) TestCancelRootRemoved() {
	ss, err := NewSession(t.root, blockdata.NewDefaultWriter(t.JSONEnc), 10)
	t.NoError(err)
	defer ss.Cancel()

	t.Equal(base.Height(10), ss.Height())
	t.NoError(ss.Cancel())

	_, err = os.Stat(t.root)
	t.True(os.IsNotExist(err))
}

func TestSession(t *testing.T) {
	suite.Run(t, new(testSession))
}
