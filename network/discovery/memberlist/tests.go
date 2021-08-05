// +build test

package memberlist

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

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
