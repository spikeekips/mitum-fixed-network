package quicnetwork

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type testPrimitiveQuicSever struct {
	suite.Suite
	bind  string
	certs []tls.Certificate
	url   *url.URL
	qn    *PrimitiveQuicServer
}

func (t *testPrimitiveQuicSever) SetupTest() {
	port, err := util.FreePort("udp")
	t.NoError(err)

	t.bind = fmt.Sprintf("localhost:%d", port)

	priv, err := util.GenerateED25519Privatekey()
	t.NoError(err)

	certs, err := util.GenerateTLSCerts(t.bind, priv)
	t.NoError(err)
	t.certs = certs

	t.url = &url.URL{Scheme: "https", Host: t.bind}
}

func (t *testPrimitiveQuicSever) readyServer(handlers map[string]network.HTTPHandlerFunc) *PrimitiveQuicServer {
	qn, err := NewPrimitiveQuicServer(t.bind, t.certs)
	t.NoError(err)

	for prefix, handler := range handlers {
		qn.SetHandler(prefix, handler)
	}

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

func (t *testPrimitiveQuicSever) TestGet() {
	handlers := map[string]network.HTTPHandlerFunc{}

	var data int = 33
	handlers["/get"] = func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(util.IntToBytes(data))
	}

	qn := t.readyServer(handlers)
	defer qn.Stop()

	client, err := NewQuicClient(true, time.Second, 1, nil)
	t.NoError(err)

	response, err := client.Request(t.url.String()+"/get", nil, nil)
	t.NoError(err)
	t.True(response.OK())

	received, err := util.BytesToInt(response.Bytes())
	t.NoError(err)
	t.Equal(data, received)
}

func (t *testPrimitiveQuicSever) TestSend() {
	handlers := map[string]network.HTTPHandlerFunc{}

	received := make(chan int, 10)
	handlers["/send"] = func(w http.ResponseWriter, r *http.Request) {
		body := &bytes.Buffer{}
		if _, err := io.Copy(body, r.Body); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)
			return
		}

		i, err := util.BytesToInt(body.Bytes())
		if err != nil {
			network.HTTPError(w, http.StatusInternalServerError)
			return
		}

		received <- i
	}

	qn := t.readyServer(handlers)
	defer qn.Stop()

	client, err := NewQuicClient(true, time.Second, 1, nil)
	t.NoError(err)

	var data int = 33
	t.NoError(client.Send(t.url.String()+"/send", util.IntToBytes(data), nil))

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("failed to receive respond"))
	case r := <-received:
		t.Equal(data, r)
	}
}

func TestPrimitiveQuicSever(t *testing.T) {
	suite.Run(t, new(testPrimitiveQuicSever))
}
