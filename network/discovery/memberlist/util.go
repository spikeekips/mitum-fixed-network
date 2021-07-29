package memberlist

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func parseHostPort(a string) (string, int, error) {
	host, uport, err := net.SplitHostPort(a)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.ParseInt(uport, 10, 64)
	if err != nil {
		return "", 0, xerrors.Errorf("wrong port, %q", a)
	}

	return host, int(port), nil
}

func stringToIPv6(s string) (net.IP, error) {
	bs := valuehash.NewSHA256([]byte(s)).Bytes()

	r := make([]string, 8)
	var sum int64
	for i := range bs {
		sum += int64(bs[i])

		if i > 0 && (i+1)%4 == 0 {
			j := int(math.Ceil(float64(i+1)/4)) - 1
			r[j] = fmt.Sprintf("%04s", strconv.FormatInt(sum, 16))
			sum = 0
		}
	}

	ip := net.ParseIP(strings.Join(r, ":"))
	if ip == nil {
		return nil, xerrors.Errorf("failed to convert to IPv6, %q", s)
	}

	return ip, nil
}

func publishToAddress(u *url.URL) (*url.URL, string, error) {
	uu := network.NormalizeURL(u)
	ip, err := stringToIPv6(uu.String())
	if err != nil {
		return nil, "", err
	}

	return uu, net.JoinHostPort(ip.String(), uu.Port()), nil
}

func isValidPublishURL(u *url.URL) error {
	if u == nil {
		return xerrors.Errorf("empty publish url")
	}
	if u.Scheme == "" {
		return xerrors.Errorf("empty publish url; empty scheme, %q", u.String())
	}
	if u.Host == "" {
		return xerrors.Errorf("empty publish url; empty host, %q", u.String())
	}
	if u.Port() == "" {
		return xerrors.Errorf("empty publish url; empty port, %q", u.String())
	}

	return nil
}

func SuffrageHandlerFilter(suffrage base.Suffrage, nodepool *network.Nodepool) func(NodeMessage) error {
	return func(msg NodeMessage) error {
		if !suffrage.IsInside(msg.Node()) {
			return xerrors.Errorf("not suffrage node, %q", msg.Node())
		}

		if !nodepool.Exists(msg.Node()) {
			return xerrors.Errorf("not in nodepool, %q", msg.Node())
		}

		return nil
	}
}
