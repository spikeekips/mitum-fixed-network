// +build test

package memberlist

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

//lint:file-ignore U1000 debugging inside test
var log logging.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	log = logging.NewLogger(&l, false)
}

type BaseDiscoveryTest struct{}

func (t *BaseDiscoveryTest) NewDiscovery(
	local *node.Local,
	connInfo network.ConnInfo,
	networkID base.NetworkID,
	enc encoder.Encoder,
	remotes map[string]http.HandlerFunc,
) *Discovery {
	req := func(
		ctx context.Context,
		insecure bool,
		timeout time.Duration,
		u,
		method string,
		body []byte,
		header http.Header,
	) (*http.Response, func() error, error) {
		handler, found := remotes[u]
		if !found {
			return nil, func() error { return nil }, xerrors.Errorf("unknown node found, %q", u)
		}
		if handler == nil {
			return nil, func() error { return nil }, xerrors.Errorf("node stopped, %q", u)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/", bytes.NewBuffer(body))
		handler(w, r)

		return w.Result(), func() error { return nil }, nil
	}

	dis := NewDiscovery(local, connInfo, networkID, enc)
	_ = dis.SetRequest(req)

	return dis
}
