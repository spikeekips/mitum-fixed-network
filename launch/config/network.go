package config

import (
	"crypto/tls"
	"net/url"
)

var (
	DefaultLocalNetworkURL  *url.URL = &url.URL{Scheme: "quic", Host: "127.0.0.1:54321"}
	DefaultLocalNetworkBind *url.URL = &url.URL{Scheme: "quic", Host: "0.0.0.0:54321"}
)

type NodeNetwork interface {
	URL() *url.URL
	SetURL(string) error
}

type BaseNodeNetwork struct {
	u *url.URL
}

func EmptyBaseNodeNetwork() *BaseNodeNetwork {
	return &BaseNodeNetwork{}
}

func (no BaseNodeNetwork) URL() *url.URL {
	return no.u
}

func (no *BaseNodeNetwork) SetURL(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else {
		no.u = u

		return nil
	}
}

type LocalNetwork interface {
	NodeNetwork
	Bind() *url.URL
	SetBind(string) error
	Certs() []tls.Certificate
	SetCerts([]tls.Certificate) error
	SetCertFiles(string /* key */, string /* cert */) error
}

type BaseLocalNetwork struct {
	*BaseNodeNetwork
	bind  *url.URL
	certs []tls.Certificate
}

func EmptyBaseLocalNetwork() *BaseLocalNetwork {
	return &BaseLocalNetwork{BaseNodeNetwork: &BaseNodeNetwork{}}
}

func (no BaseLocalNetwork) Bind() *url.URL {
	return no.bind
}

func (no *BaseLocalNetwork) SetBind(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else {
		no.bind = u

		return nil
	}
}

func (no BaseLocalNetwork) Certs() []tls.Certificate {
	return no.certs
}

func (no *BaseLocalNetwork) SetCerts(certs []tls.Certificate) error {
	no.certs = certs

	return nil
}

func (no *BaseLocalNetwork) SetCertFiles(key, cert string) error {
	if c, err := tls.LoadX509KeyPair(key, cert); err != nil {
		return err
	} else {
		no.certs = []tls.Certificate{c}

		return nil
	}
}
