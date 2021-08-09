package network

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
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
		return errors.Wrap(err, "failed to open bind")
	case <-time.After(timeout):
		return nil
	}
}

func ParseURL(s string, allowEmpty bool) (*url.URL, error) { // nolint:unparam
	s = strings.TrimSpace(s)
	if len(s) < 1 {
		if !allowEmpty {
			return nil, errors.Errorf("empty url string")
		}

		return nil, nil
	}

	return url.Parse(s)
}

func NormalizeURLString(s string) (*url.URL, error) {
	u, err := ParseURL(s, false)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid url, %q", s)
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
		return errors.Errorf("empty url")
	}
	if u.Scheme == "" {
		return errors.Errorf("empty scheme, %q", u.String())
	}

	switch {
	case u.Host == "":
		return errors.Errorf("empty host, %q", u.String())
	case strings.HasPrefix(u.Host, ":") && u.Host == fmt.Sprintf(":%s", u.Port()):
		return errors.Errorf("empty host, %q", u.String())
	}

	return nil
}

// ParseCombinedNodeURL parses the combined url of node; it contains,
// - node publish url
// - tls insecure: "#insecure"
// "insecure" fragment will be removed.
func ParseCombinedNodeURL(u *url.URL) (*url.URL, bool, error) {
	if err := IsValidURL(u); err != nil {
		return nil, false, errors.Wrap(err, "invalid combined node url")
	}

	i := NormalizeURL(u)

	insecure := i.Fragment == "insecure"
	i.Fragment = ""

	return i, insecure, nil
}
