package quicnetwork

import (
	"net/http"
	"sync"

	"github.com/lucas-clemente/quic-go/http3"
)

var httpClientPool = sync.Pool{
	New: func() interface{} {
		return new(http.Client)
	},
}

var roundTripperPool = sync.Pool{
	New: func() interface{} {
		return new(http3.RoundTripper)
	},
}

var (
	HTTPClientPoolGet = func() *http.Client {
		return httpClientPool.Get().(*http.Client)
	}
	HTTPClientPoolPut = func(c *http.Client) {
		c.Transport = nil
		c.CheckRedirect = nil
		c.Jar = nil
		c.Timeout = 0

		httpClientPool.Put(c)
	}

	RoundTripperPoolGet = func() *http3.RoundTripper {
		return roundTripperPool.Get().(*http3.RoundTripper)
	}
	RoundTripperPoolPut = func(r *http3.RoundTripper) {
		// NOTE RoundTripper should be closed by Close()
		r.DisableCompression = false
		r.TLSClientConfig = nil
		r.QuicConfig = nil
		r.Dial = nil
		r.MaxResponseHeaderBytes = 0

		roundTripperPool.Put(r)
	}
)
