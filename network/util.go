package network

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

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

func ParseURL(s string, allowEmpty bool) (*url.URL, error) { // nolint:unparam
	s = strings.TrimSpace(s)
	if len(s) < 1 {
		if !allowEmpty {
			return nil, xerrors.Errorf("empty url string")
		}

		return nil, nil
	}

	return url.Parse(s)
}

func NormalizeURLString(s string) (*url.URL, error) {
	u, err := ParseURL(s, false)
	if err != nil {
		return nil, xerrors.Errorf("invalid url, %q: %w", s, err)
	}

	return NormalizeURL(u), nil
}

func NormalizeURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}

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

	return uu
}

func IsValidURL(u *url.URL) error {
	if u == nil {
		return xerrors.Errorf("empty url")
	}
	if u.Scheme == "" {
		return xerrors.Errorf("empty scheme, %q", u.String())
	}

	switch {
	case u.Host == "":
		return xerrors.Errorf("empty host, %q", u.String())
	case strings.HasPrefix(u.Host, ":") && u.Host == fmt.Sprintf(":%s", u.Port()):
		return xerrors.Errorf("empty host, %q", u.String())
	}

	return nil
}

// ParseCombinedNodeURL parses the combined url of node; it contains,
// - node publish url
// - tls insecure: "#insecure"
// "insecure" fragment will be removed.
func ParseCombinedNodeURL(u *url.URL) (*url.URL, bool, error) {
	if err := IsValidURL(u); err != nil {
		return nil, false, xerrors.Errorf("invalid combined node url: %w", err)
	}

	i := NormalizeURL(u)

	insecure := i.Fragment == "insecure"
	i.Fragment = ""

	return i, insecure, nil
}
