package quicnetwork

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testQuicServer struct {
	suite.Suite
	encs     *encoder.Encoders
	enc      encoder.Encoder
	bind     string
	certs    []tls.Certificate
	connInfo network.HTTPConnInfo
}

func (t *testQuicServer) SetupTest() {
	t.encs = encoder.NewEncoders()
	t.enc = jsonenc.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)
	_ = t.encs.TestAddHinter(ballot.ProposalFactHinter)
	_ = t.encs.TestAddHinter(ballot.ProposalHinter)
	_ = t.encs.TestAddHinter(base.BallotFactSignHinter)
	_ = t.encs.TestAddHinter(base.SignedBallotFactHinter)
	_ = t.encs.TestAddHinter(base.DummyVoteproof{})
	_ = t.encs.TestAddHinter(base.StringAddressHinter)
	_ = t.encs.TestAddHinter(block.BaseBlockDataMapHinter)
	_ = t.encs.TestAddHinter(block.ManifestV0Hinter)
	_ = t.encs.TestAddHinter(key.BasePrivatekey{})
	_ = t.encs.TestAddHinter(key.BasePublickey{})
	_ = t.encs.TestAddHinter(network.EndHandoverSealV0Hinter)
	_ = t.encs.TestAddHinter(network.HTTPConnInfoHinter)
	_ = t.encs.TestAddHinter(network.NodeInfoV0Hinter)
	_ = t.encs.TestAddHinter(network.PingHandoverSealV0Hinter)
	_ = t.encs.TestAddHinter(node.BaseV0Hinter)
	_ = t.encs.TestAddHinter(seal.DummySeal{})
	_ = t.encs.TestAddHinter(state.BytesValueHinter)
	_ = t.encs.TestAddHinter(state.StateV0{})
	_ = t.encs.TestAddHinter(operation.KVOperationFact{})
	_ = t.encs.TestAddHinter(operation.KVOperation{})
	_ = t.encs.TestAddHinter(base.BaseFactSignHinter)

	port, err := util.FreePort("udp")
	t.NoError(err)

	t.bind = fmt.Sprintf("localhost:%d", port)

	priv, err := util.GenerateED25519Privatekey()
	t.NoError(err)

	certs, err := util.GenerateTLSCerts(t.bind, priv)
	t.NoError(err)
	t.certs = certs

	u, err := network.NormalizeURLString(fmt.Sprintf("https://%s", t.bind))
	t.NoError(err)
	t.connInfo = network.NewHTTPConnInfo(u, true)
}

func (t *testQuicServer) readyServer() *Server {
	qs, err := NewPrimitiveQuicServer(t.bind, t.certs, nil)
	t.NoError(err)

	ca, err := cache.NewGCache("lru", 100, time.Second*3)
	t.NoError(err)

	qn, err := NewServer(qs, t.encs, t.enc, ca, t.connInfo, nil)
	t.NoError(err)

	t.NoError(qn.Start())

	_, port, err := net.SplitHostPort(t.bind)
	t.NoError(err)

	maxRetries := 3
	var retries int
	for {
		if retries == maxRetries {
			t.NoError(errors.Errorf("quic server did not respond"))
			break
		}

		if err := util.CheckPort("udp", fmt.Sprintf("127.0.0.1:%s", port), time.Millisecond*50); err == nil {
			break
		}
		<-time.After(time.Millisecond * 10)
		retries++
	}

	return qn
}

func (t *testQuicServer) TestNew() {
	qs, err := NewPrimitiveQuicServer(t.bind, t.certs, nil)
	t.NoError(err)

	qn, err := NewServer(qs, t.encs, t.enc, nil, t.connInfo, nil)
	t.NoError(err)

	t.Implements((*network.Server)(nil), qn)
	t.IsType(cache.Dummy{}, qn.cache)
}

func (t *testQuicServer) TestSendSeal() {
	qn := t.readyServer()
	defer qn.Stop()

	received := make(chan seal.Seal, 10)
	qn.SetNewSealHandler(func(sl seal.Seal) error {
		received <- sl
		return nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)
	t.Implements((*network.Channel)(nil), qc)

	sl := seal.NewDummySeal(key.NewBasePrivatekey().Publickey())

	t.NoError(qc.SendSeal(context.TODO(), nil, sl))

	select {
	case <-time.After(time.Second):
		t.NoError(errors.Errorf("failed to receive respond"))
	case r := <-received:
		t.Equal(sl.Hint(), r.Hint())
		t.True(sl.Hash().Equal(r.Hash()))
		t.True(sl.BodyHash().Equal(r.BodyHash()))
		t.True(sl.Signer().Equal(r.Signer()))
		t.Equal(sl.Signature(), r.Signature())
		t.True(localtime.Equal(sl.SignedAt(), r.SignedAt()))
	}

	t.NoError(qc.SendSeal(context.TODO(), nil, sl))
}

func (t *testQuicServer) TestGetStagedOperations() {
	qn := t.readyServer()
	defer qn.Stop()

	var hs []valuehash.Hash
	ops := map[string]operation.Operation{}
	for i := 0; i < 3; i++ {
		op, err := operation.NewKVOperation(key.NewBasePrivatekey(), util.UUID().Bytes(), util.UUID().String(), util.UUID().Bytes(), nil)
		t.NoError(err)

		ops[op.Fact().Hash().String()] = op
		hs = append(hs, op.Fact().Hash())
	}

	qn.SetGetStagedOperationsHandler(func(hs []valuehash.Hash) ([]operation.Operation, error) {
		var l []operation.Operation

		for _, ih := range hs {
			h := ih.(valuehash.Bytes)
			if op, found := ops[h.String()]; found {
				l = append(l, op)
			}
		}

		return l, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	{ // get all
		l, err := qc.StagedOperations(context.TODO(), hs)
		t.NoError(err)
		t.Equal(len(hs), len(l))

		sm := map[string]operation.Operation{}
		for _, s := range l {
			sm[s.Fact().Hash().String()] = s
		}

		for h, op := range ops {
			t.True(op.Fact().Hash().Equal(sm[h].Fact().Hash()))
			t.True(op.Hash().Equal(sm[h].Hash()))
		}
	}

	{ // some of them
		l, err := qc.StagedOperations(context.TODO(), hs[:2])
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[string]operation.Operation{}
		for _, s := range l {
			sm[s.Fact().Hash().String()] = s
		}

		for _, h := range hs[:2] {
			t.True(ops[h.String()].Hash().Equal(sm[h.String()].Hash()))
			t.True(ops[h.String()].Fact().Hash().Equal(sm[h.String()].Fact().Hash()))
		}
	}

	{ // with unknown
		bad := hs[:2]
		bad = append(bad, valuehash.RandomSHA256())

		l, err := qc.StagedOperations(context.TODO(), bad)
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[string]operation.Operation{}
		for _, s := range l {
			sm[s.Fact().Hash().String()] = s
		}

		for _, h := range hs[:2] {
			t.True(ops[h.String()].Hash().Equal(sm[h.String()].Hash()))
			t.True(ops[h.String()].Fact().Hash().Equal(sm[h.String()].Fact().Hash()))
		}
	}
}

func (t *testQuicServer) TestGetProposal() {
	qn := t.readyServer()
	defer qn.Stop()

	hashes := make([]valuehash.Hash, 4)
	proposals := map[string]base.Proposal{}

	for i := 0; i < 4; i++ {
		fact := ballot.NewProposalFact(
			base.Height(33),
			base.Round(0),
			base.RandomStringAddress(),
			[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()},
		)
		bvp := base.NewDummyVoteproof(
			fact.Height(),
			fact.Round(),
			base.StageINIT,
			base.VoteResultMajority,
		)

		pr, err := ballot.NewProposal(fact, fact.Proposer(), bvp, key.NewBasePrivatekey(), nil)
		t.NoError(err)

		proposals[fact.Hash().String()] = pr
		hashes[i] = fact.Hash()
	}

	qn.SetGetProposalHandler(func(h valuehash.Hash) (base.Proposal, error) {
		pr, found := proposals[h.String()]
		if !found {
			return nil, nil
		}

		return pr, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	{ // normal
		h := hashes[0]
		pr, err := qc.Proposal(context.TODO(), h)
		t.NoError(err)
		t.NoError(pr.IsValid(nil))

		t.True(h.Equal(pr.Fact().Hash()))

		asfs := proposals[pr.Fact().Hash().String()]

		afact := asfs.Fact()
		afs := asfs.FactSign()
		bfact := pr.Fact()
		bfs := pr.FactSign()

		t.Equal(bfact.Stage(), base.StageProposal)
		t.True(afact.Hash().Equal(bfact.Hash()))
		t.True(afs.Signer().Equal(bfs.Signer()))
	}

	{ // unknown
		_, err := qc.Proposal(context.TODO(), valuehash.RandomSHA256())
		t.True(errors.Is(err, util.NotFoundError))
	}
}

func (t *testQuicServer) TestNodeInfo() {
	qn := t.readyServer()
	defer qn.Stop()

	nid := []byte("test-network-id")

	var ni network.NodeInfo
	{
		blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
		t.NoError(err)

		suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)

		ni = network.NewNodeInfoV0(
			node.RandomNode("n0"),
			nid,
			base.StateBooting,
			blk.Manifest(),
			util.Version("0.1.1"),
			map[string]interface{}{"showme": 1.1},
			nil,
			suffrage,
			t.connInfo,
		)
	}

	qn.SetNodeInfoHandler(func() (network.NodeInfo, error) {
		return ni, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	nni, err := qc.NodeInfo(context.TODO())
	t.NoError(err)

	network.CompareNodeInfo(t.T(), ni, nni)
}

func (t *testQuicServer) TestEmptyBlockDataMaps() {
	qn := t.readyServer()
	defer qn.Stop()

	qn.SetBlockDataMapsHandler(func(hs []base.Height) ([]block.BlockDataMap, error) {
		return nil, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	bds, err := qc.BlockDataMaps(context.TODO(), []base.Height{33, 34})
	t.NoError(err)

	t.Empty(bds)
}

func (t *testQuicServer) TestBlockDataMaps() {
	qn := t.readyServer()
	defer qn.Stop()

	bd := block.NewBaseBlockDataMap(block.TestBlockDataWriterHint, 33)
	bd = bd.SetBlock(valuehash.RandomSHA256())

	for _, k := range block.BlockData {
		bd, _ = bd.SetItem(block.NewBaseBlockDataMapItem(k, util.UUID().String(), "file://"+util.UUID().String()))
	}
	{
		i, err := bd.UpdateHash()
		t.NoError(err)
		bd = i
	}

	qn.SetBlockDataMapsHandler(func(hs []base.Height) ([]block.BlockDataMap, error) {
		return []block.BlockDataMap{
			bd,
		}, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	bds, err := qc.BlockDataMaps(context.TODO(), []base.Height{33, 34})
	t.NoError(err)

	t.Equal(1, len(bds))

	block.CompareBlockDataMap(t.Assert(), bd, bds[0])
}

func (t *testQuicServer) TestEmptyBlockData() {
	qn := t.readyServer()
	defer qn.Stop()

	qn.SetBlockDataHandler(func(p string) (io.Reader, func() error, error) {
		return nil, func() error { return nil }, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	item := block.NewBaseBlockDataMapItem("findme", util.UUID().String(), "file:///showme/findme")
	_, err = qc.BlockData(context.Background(), item)
	t.Contains(err.Error(), "failed to request")
}

func (t *testQuicServer) TestGetBlockDataWithError() {
	qn := t.readyServer()
	defer qn.Stop()

	qn.SetBlockDataHandler(func(p string) (io.Reader, func() error, error) {
		return nil, func() error { return nil }, util.NotFoundError
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	item := block.NewBaseBlockDataMapItem("findme", util.UUID().String(), "file:///showme/findme")
	_, err = qc.BlockData(context.Background(), item)
	t.Contains(err.Error(), "not found")
}

func (t *testQuicServer) TestGetBlockData() {
	qn := t.readyServer()
	defer qn.Stop()

	f, err := ioutil.TempFile("", "")
	t.NoError(err)

	data := []byte("findme")
	f.Write(data)
	_ = f.Close()

	checksum, err := util.GenerateFileChecksum(f.Name())
	t.NoError(err)

	f, err = os.Open(f.Name())
	t.NoError(err)

	defer func() {
		os.Remove(f.Name())
	}()

	qn.SetBlockDataHandler(func(p string) (io.Reader, func() error, error) {
		return f, f.Close, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	item := block.NewBaseBlockDataMapItem("findme", checksum, "file:///showme/findme")
	r, err := qc.BlockData(context.Background(), item)
	t.NoError(err)
	t.NotNil(r)

	defer r.Close()

	b, err := io.ReadAll(r)
	t.NoError(err)
	t.Equal(data, b)
}

func (t *testQuicServer) TestPassthroughs() {
	qn := t.readyServer()
	defer qn.Stop()

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)
	t.Implements((*network.Channel)(nil), qc)

	// attach Nodepool
	local := node.RandomLocal("local")
	ns := network.NewNodepool(local, qc)

	ch0 := network.NilConnInfoChannel("n0")
	t.NoError(ns.SetPassthrough(ch0, nil, 0))

	passedch := make(chan seal.Seal, 10)
	ch0.SetNewSealHandler(func(sl seal.Seal) error {
		passedch <- sl
		return nil
	})

	qn.passthroughs = ns.Passthroughs

	sl := seal.NewDummySeal(key.NewBasePrivatekey().Publickey())

	t.NoError(qc.SendSeal(context.TODO(), t.connInfo, sl))

	select {
	case <-time.After(time.Second):
		t.NoError(errors.Errorf("failed to receive respond"))
	case r := <-passedch:
		t.Equal(sl.Hint(), r.Hint())
		t.True(sl.Hash().Equal(r.Hash()))
		t.True(sl.BodyHash().Equal(r.BodyHash()))
		t.True(sl.Signer().Equal(r.Signer()))
		t.Equal(sl.Signature(), r.Signature())
		t.True(localtime.Equal(sl.SignedAt(), r.SignedAt()))
	}
}

func (t *testQuicServer) TestPassthroughsFilterFrom() {
	qn := t.readyServer()
	defer qn.Stop()

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)
	t.Implements((*network.Channel)(nil), qc)

	// attach Nodepool
	local := node.RandomLocal("local")
	ns := network.NewNodepool(local, qc)

	ch0 := network.NilConnInfoChannel("n0")
	t.NoError(ns.SetPassthrough(ch0, nil, 0))

	passedch := make(chan seal.Seal, 10)
	ch0.SetNewSealHandler(func(sl seal.Seal) error {
		passedch <- sl
		return nil
	})

	qn.passthroughs = ns.Passthroughs

	sl := seal.NewDummySeal(key.NewBasePrivatekey().Publickey())

	t.NoError(qc.SendSeal(context.TODO(), ch0.ConnInfo(), sl))

	select {
	case <-time.After(time.Second):
	case <-passedch:
		t.NoError(errors.Errorf("seal should be filtered"))
	}
}

func (t *testQuicServer) TestHandoverHandlers() {
	qn := t.readyServer()
	defer qn.Stop()

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)
	t.Implements((*network.Channel)(nil), qc)

	newconnInfo := network.NewHTTPConnInfo(&url.URL{Scheme: "https", Host: "new"}, true)

	compareSeal := func(a, b seal.Seal) {
		t.Equal(a.Hint(), b.Hint())
		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.BodyHash().Equal(b.BodyHash()))
		t.True(a.Signer().Equal(b.Signer()))
		t.Equal(a.Signature(), b.Signature())
		t.True(localtime.Equal(a.SignedAt(), b.SignedAt()))
	}

	t.Run("ping-handover", func() {
		receivedch := make(chan seal.Seal, 10)
		qn.SetPingHandoverHandler(func(sl network.PingHandoverSeal) (bool, error) {
			receivedch <- sl
			return true, nil
		})

		sl, err := network.NewHandoverSealV0(network.PingHandoverSealV0Hint, key.NewBasePrivatekey(), base.RandomStringAddress(), newconnInfo, nil)
		t.NoError(err)

		ok, err := qc.PingHandover(context.TODO(), sl)
		t.NoError(err)
		t.True(ok)

		select {
		case <-time.After(time.Second):
			t.NoError(errors.Errorf("failed to receive respond"))
		case r := <-receivedch:
			t.Equal(sl.Hint(), r.Hint())
			compareSeal(sl, r)
		}
	})

	t.Run("end-handover", func() {
		receivedch := make(chan seal.Seal, 10)
		qn.SetEndHandoverHandler(func(sl network.EndHandoverSeal) (bool, error) {
			receivedch <- sl
			return true, nil
		})

		sl, err := network.NewHandoverSealV0(network.EndHandoverSealV0Hint, key.NewBasePrivatekey(), base.RandomStringAddress(), newconnInfo, nil)
		t.NoError(err)

		ok, err := qc.EndHandover(context.TODO(), sl)
		t.NoError(err)
		t.True(ok)

		select {
		case <-time.After(time.Second):
			t.NoError(errors.Errorf("failed to receive respond"))
		case r := <-receivedch:
			t.Equal(sl.Hint(), r.Hint())
			compareSeal(sl, r)
		}
	})

	t.Run("ping-handover-not-ok", func() {
		qn.SetPingHandoverHandler(func(sl network.PingHandoverSeal) (bool, error) {
			return false, nil
		})

		sl, err := network.NewHandoverSealV0(network.PingHandoverSealV0Hint, key.NewBasePrivatekey(), base.RandomStringAddress(), newconnInfo, nil)
		t.NoError(err)

		ok, err := qc.PingHandover(context.TODO(), sl)
		t.NoError(err)
		t.False(ok)
	})

	t.Run("end-handover-not-ok", func() {
		qn.SetEndHandoverHandler(func(sl network.EndHandoverSeal) (bool, error) {
			return false, nil
		})

		sl, err := network.NewHandoverSealV0(network.EndHandoverSealV0Hint, key.NewBasePrivatekey(), base.RandomStringAddress(), newconnInfo, nil)
		t.NoError(err)

		ok, err := qc.EndHandover(context.TODO(), sl)
		t.NoError(err)
		t.False(ok)
	})
}

func TestQuicServer(t *testing.T) {
	suite.Run(t, new(testQuicServer))
}
