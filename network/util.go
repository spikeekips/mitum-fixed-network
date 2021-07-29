package network

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func CheckBindIsOpen(network, bind string, timeout time.Duration) error {
	errchan := make(chan error)
	switch network {
	case "tcp":
		go func() {
			if server, err := net.Listen(network, bind); err != nil {
				errchan <- err
			} else if server != nil {
				_ = server.Close()
			}
		}()
	case "udp":
		go func() {
			if server, err := net.ListenPacket(network, bind); err != nil {
				errchan <- err
			} else if server != nil {
				_ = server.Close()
			}
		}()
	}

	select {
	case err := <-errchan:
		return xerrors.Errorf("failed to open bind: %w", err)
	case <-time.After(timeout):
		return nil
	}
}

func NormalizeURLString(s string) (*url.URL, error) {
	if len(strings.TrimSpace(s)) < 1 {
		return nil, xerrors.Errorf("empty url")
	}

	u, err := url.Parse(s)
	if err != nil {
		return nil, xerrors.Errorf("invalid url, %q: %w", s, err)
	}

	return NormalizeURL(u), nil
}

func NormalizeURL(u *url.URL) *url.URL {
	uu := &url.URL{
		Scheme:      u.Scheme,
		Opaque:      u.Opaque,
		User:        u.User,
		Host:        u.Host,
		Path:        u.Path,
		RawPath:     u.RawPath,
		ForceQuery:  u.ForceQuery,
		RawQuery:    u.RawQuery,
		Fragment:    u.Fragment,
		RawFragment: u.RawFragment,
	}

	if port := uu.Port(); port == "" {
		switch uu.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			port = "0"
		}
		uu.Host = uu.Hostname() + ":" + port
	}

	if uu.Path == "/" {
		uu.Path = ""
	}

	uu.Fragment = ""

	return uu
}

func NormalizeNodeURL(s string) (HTTPConnInfo, error) {
	u, err := NormalizeURLString(s)
	if err != nil {
		return HTTPConnInfo{}, xerrors.Errorf("wrong node url, %q: %w", s, err)
	}

	query := u.Query()
	insecure := util.ParseBoolInQuery(query.Get("insecure"))
	query.Del("insecure")

	u.RawQuery = query.Encode()

	return NewHTTPConnInfo(u, insecure), nil
}
