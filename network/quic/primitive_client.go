package quicnetwork

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type clientDoRequestFunc func(context.Context, time.Duration, string, []byte, http.Header) (*QuicResponse, error)

type QuicClient struct {
	*logging.Logging
	insecure   bool
	quicConfig *quic.Config
}

func NewQuicClient(insecure bool, quicConfig *quic.Config) (*QuicClient, error) {
	if quicConfig == nil {
		quicConfig = &quic.Config{}
	}

	if quicConfig.HandshakeTimeout < 1 {
		quicConfig.HandshakeTimeout = time.Second * 3
	}

	if quicConfig.MaxIdleTimeout < 1 {
		quicConfig.MaxIdleTimeout = time.Second * 30 // long enough
	}

	return &QuicClient{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-client")
		}),
		insecure:   insecure,
		quicConfig: quicConfig,
	}, nil
}

func (cl *QuicClient) Request(
	ctx context.Context, timeout time.Duration,
	url string, b []byte, headers http.Header,
) (*QuicResponse, error) {
	client, closefunc := cl.newClient(timeout)
	if res, err := cl.request(ctx, client, url, "GET", b, headers); err != nil {
		defer func() {
			_ = closefunc()
		}()

		return nil, err
	} else {
		return NewQuicResponse(res, closefunc), nil
	}
}

func (cl *QuicClient) Send(
	ctx context.Context, timeout time.Duration,
	url string, b []byte, headers http.Header,
) (*QuicResponse, error) {
	client, closefunc := cl.newClient(timeout)
	if res, err := cl.request(ctx, client, url, "POST", b, headers); err != nil {
		defer func() {
			_ = closefunc()
		}()

		return nil, err
	} else {
		return NewQuicResponse(res, closefunc), nil
	}
}

func (cl *QuicClient) request(
	ctx context.Context,
	client *http.Client,
	url string,
	method string,
	b []byte,
	headers http.Header,
) (*http.Response, error) {
	if i, err := cl.makeRequest(url, method, b, headers); err != nil {
		return nil, err
	} else {
		return client.Do(i.WithContext(ctx))
	}
}

func (cl *QuicClient) makeRequest(url string, method string, b []byte, headers http.Header) (*http.Request, error) {
	l := cl.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("url", url).
			Int("content_length", len(b)).
			Str("method", method).
			Interface("headers", headers).
			Str("request", "request")
	})

	var request *http.Request
	{
		var err error
		switch method {
		case "GET":
			request, err = http.NewRequest("GET", url, nil)
		case "POST":
			request, err = http.NewRequest("POST", url, bytes.NewBuffer(b))
		}

		if err != nil {
			l.Error().Err(err).Msg("failed to create request")

			return nil, err
		}
	}

	request.Header = headers

	return request, nil
}

func (cl *QuicClient) newClient(maxIdleTimeout time.Duration) (*http.Client, func() error /* close func */) {
	qcconfig := CloneConfig(cl.quicConfig)
	if maxIdleTimeout > 0 {
		qcconfig.MaxIdleTimeout = maxIdleTimeout
	}

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cl.insecure, // nolint
		},
		QuicConfig: qcconfig,
	}

	return &http.Client{Transport: roundTripper}, roundTripper.Close
}

type QuicResponse struct {
	sync.Mutex
	*http.Response
	closeFunc func() error
	body      io.Reader
}

func NewQuicResponse(response *http.Response, closeFunc func() error) *QuicResponse {
	return &QuicResponse{Response: response, closeFunc: closeFunc}
}

func (qr *QuicResponse) OK() bool {
	return qr.StatusCode == 200 || qr.StatusCode == 201
}

func (qr *QuicResponse) Bytes() ([]byte, error) {
	qr.Lock()
	defer qr.Unlock()

	if qr.body == nil {
		body := &bytes.Buffer{}
		if _, err := io.Copy(body, qr.Response.Body); err != nil {
			return nil, err
		}

		qr.body = body
	}

	return qr.body.(*bytes.Buffer).Bytes(), nil
}

func (qr *QuicResponse) Error() error {
	if qr.OK() {
		return nil
	} else if qr.StatusCode == http.StatusNotFound {
		return storage.NotFoundError.Errorf("request not found: %d", qr.StatusCode)
	}

	return xerrors.Errorf("failed to request: %d", qr.StatusCode)
}

func (qr *QuicResponse) Close() error {
	_ = qr.Response.Body.Close()

	return qr.closeFunc()
}

func (qr *QuicResponse) Body() io.ReadCloser {
	qr.Lock()
	defer qr.Unlock()

	if qr.body != nil {
		return util.NewNilReadCloser(qr.body)
	}

	return qr.Response.Body
}

func CloneConfig(c *quic.Config) *quic.Config {
	cp := *c

	return &cp
}
