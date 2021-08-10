package quicnetwork

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
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
	_ = t.encs.TestAddHinter(key.BTCPrivatekeyHinter)
	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(seal.DummySeal{})
	_ = t.encs.TestAddHinter(node.BaseV0{})
	_ = t.encs.TestAddHinter(base.StringAddress(""))
	_ = t.encs.TestAddHinter(block.ManifestV0{})
	_ = t.encs.TestAddHinter(network.NodeInfoV0{})
	_ = t.encs.TestAddHinter(state.StateV0{})
	_ = t.encs.TestAddHinter(state.BytesValue{})
	_ = t.encs.TestAddHinter(block.BaseBlockDataMap{})

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

	qn, err := NewServer(qs, t.encs, t.enc, ca)
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

	qn, err := NewServer(qs, t.encs, t.enc, nil)
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

	sl := seal.NewDummySeal(key.MustNewBTCPrivatekey())

	t.NoError(qc.SendSeal(context.TODO(), sl))

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

	// NOTE if already known seal received, server returns 200
	qn.SetHasSealHandler(func(h valuehash.Hash) (bool, error) {
		return true, nil
	})

	t.NoError(qc.SendSeal(context.TODO(), sl))
}

func (t *testQuicServer) TestGetSeals() {
	qn := t.readyServer()
	defer qn.Stop()

	var hs []valuehash.Hash
	seals := map[string]seal.Seal{}
	for i := 0; i < 3; i++ {
		sl := seal.NewDummySeal(key.MustNewBTCPrivatekey())

		seals[sl.Hash().String()] = sl
		hs = append(hs, sl.Hash())
	}

	qn.SetGetSealsHandler(func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var sls []seal.Seal

		for _, ih := range hs {
			h := ih.(valuehash.Bytes)
			if sl, found := seals[h.String()]; found {
				sls = append(sls, sl)
			}
		}

		return sls, nil
	})

	qc, err := NewChannel(t.connInfo, 2, nil, t.encs, t.enc)
	t.NoError(err)

	{ // get all
		l, err := qc.Seals(context.TODO(), hs)
		t.NoError(err)
		t.Equal(len(hs), len(l))

		sm := map[string]seal.Seal{}
		for _, s := range l {
			sm[s.Hash().String()] = s
		}

		for h, sl := range seals {
			t.True(sl.Hash().Equal(sm[h].Hash()))
		}
	}

	{ // some of them
		l, err := qc.Seals(context.TODO(), hs[:2])
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[string]seal.Seal{}
		for _, s := range l {
			sm[s.Hash().String()] = s
		}

		for _, h := range hs[:2] {
			t.True(seals[h.String()].Hash().Equal(sm[h.String()].Hash()))
		}
	}

	{ // with unknown
		bad := hs[:2]
		bad = append(bad, valuehash.RandomSHA256())

		l, err := qc.Seals(context.TODO(), bad)
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[string]seal.Seal{}
		for _, s := range l {
			sm[s.Hash().String()] = s
		}

		for _, h := range hs[:2] {
			t.True(seals[h.String()].Hash().Equal(sm[h.String()].Hash()))
		}
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
			"https://local",
			map[string]interface{}{"showme": 1.1},
			nil,
			suffrage,
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

func TestQuicServer(t *testing.T) {
	suite.Run(t, new(testQuicServer))
}
