package quicnetwork

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type testQuicSever struct {
	suite.Suite
	encs  *encoder.Encoders
	enc   encoder.Encoder
	bind  string
	certs []tls.Certificate
	url   *url.URL
	qn    *QuicServer
}

func (t *testQuicSever) SetupTest() {
	t.encs = encoder.NewEncoders()
	t.enc = jsonencoder.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)
	_ = t.encs.AddHinter(key.BTCPrivatekey{})
	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(seal.DummySeal{})

	port, err := util.FreePort("udp")
	t.NoError(err)

	t.bind = fmt.Sprintf("localhost:%d", port)

	priv, err := util.GenerateED25519Privatekey()
	t.NoError(err)

	certs, err := util.GenerateTLSCerts(t.bind, priv)
	t.NoError(err)
	t.certs = certs

	t.url = &url.URL{Scheme: "quic", Host: t.bind}
}

func (t *testQuicSever) readyServer() *QuicServer {
	qs, err := NewPrimitiveQuicServer(t.bind, t.certs)
	t.NoError(err)

	qn, err := NewQuicServer(qs, t.encs, t.enc)
	t.NoError(err)

	t.NoError(qn.Start())

	_, port, err := net.SplitHostPort(t.bind)
	t.NoError(err)

	maxRetries := 3
	var retries int
	for {
		if retries == maxRetries {
			t.NoError(xerrors.Errorf("quic server did not respond"))
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

func (t *testQuicSever) TestSendSeal() {
	qn := t.readyServer()
	defer qn.Stop()

	received := make(chan seal.Seal, 10)
	qn.SetNewSealHandler(func(sl seal.Seal) error {
		received <- sl
		return nil
	})

	qc, err := NewQuicChannel(t.url.String(), 2, true, time.Millisecond*500, 3, nil, t.encs, t.enc)
	t.NoError(err)

	sl := seal.NewDummySeal(key.MustNewBTCPrivatekey())

	t.NoError(qc.SendSeal(sl))

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("failed to receive respond"))
	case r := <-received:
		t.Equal(sl.Hint(), r.Hint())
		t.Equal(sl.Hash(), r.Hash())
		t.Equal(sl.BodyHash(), r.BodyHash())
		t.Equal(sl.Signer(), r.Signer())
		t.Equal(sl.Signature(), r.Signature())
		t.Equal(localtime.RFC3339(sl.SignedAt()), localtime.RFC3339(r.SignedAt()))
	}
}

func (t *testQuicSever) TestGetSeals() {
	qn := t.readyServer()
	defer qn.Stop()

	var hs []valuehash.Hash
	seals := map[valuehash.Hash]seal.Seal{}
	for i := 0; i < 3; i++ {
		sl := seal.NewDummySeal(key.MustNewBTCPrivatekey())

		seals[sl.Hash()] = sl
		hs = append(hs, sl.Hash())
	}

	qn.SetGetSealsHandler(func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var sls []seal.Seal

		for _, h := range hs {
			if sl, found := seals[h]; found {
				sls = append(sls, sl)
			}
		}

		return sls, nil
	})

	qc, err := NewQuicChannel(t.url.String(), 2, true, time.Millisecond*500, 3, nil, t.encs, t.enc)
	t.NoError(err)

	{ // get all
		l, err := qc.Seals(hs)
		t.NoError(err)
		t.Equal(len(hs), len(l))

		sm := map[valuehash.Hash]seal.Seal{}
		for _, s := range l {
			sm[s.Hash()] = s
		}

		for h, sl := range seals {
			t.True(sl.Hash().Equal(sm[h].Hash()))
		}
	}

	{ // some of them
		l, err := qc.Seals(hs[:2])
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[valuehash.Hash]seal.Seal{}
		for _, s := range l {
			sm[s.Hash()] = s
		}

		for _, h := range hs[:2] {
			t.True(seals[h].Hash().Equal(sm[h].Hash()))
		}
	}

	{ // with unknown
		bad := hs[:2]
		bad = append(bad, valuehash.RandomSHA256())

		l, err := qc.Seals(bad)
		t.NoError(err)
		t.Equal(len(hs[:2]), len(l))

		sm := map[valuehash.Hash]seal.Seal{}
		for _, s := range l {
			sm[s.Hash()] = s
		}

		for _, h := range hs[:2] {
			t.True(seals[h].Hash().Equal(sm[h].Hash()))
		}
	}
}

func TestQuicSever(t *testing.T) {
	suite.Run(t, new(testQuicSever))
}
