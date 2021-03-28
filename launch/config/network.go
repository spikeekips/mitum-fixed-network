package config

import (
	"crypto/tls"
	"net/url"

	"github.com/spikeekips/mitum/util/cache"
)

var (
	DefaultLocalNetworkURL       *url.URL = &url.URL{Scheme: "quic", Host: "127.0.0.1:54321"}
	DefaultLocalNetworkBind      *url.URL = &url.URL{Scheme: "quic", Host: "0.0.0.0:54321"}
	DefaultLocalNetworkCache              = "gcache:?type=lru&size=100&expire=3s"
	DefaultLocalNetworkSealCache          = "gcache:?type=lru&size=10000&expire=3m"
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
	Cache() *url.URL
	SetCache(string) error
	SealCache() *url.URL
	SetSealCache(string) error
}

type BaseLocalNetwork struct {
	*BaseNodeNetwork
	bind      *url.URL
	certs     []tls.Certificate
	cache     *url.URL
	sealCache *url.URL
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

func (no BaseLocalNetwork) Cache() *url.URL {
	return no.cache
}

func (no *BaseLocalNetwork) SetCache(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else if _, err := cache.NewCacheFromURI(u.String()); err != nil {
		return err
	} else {
		no.cache = u

		return nil
	}
}

func (no BaseLocalNetwork) SealCache() *url.URL {
	return no.sealCache
}

func (no *BaseLocalNetwork) SetSealCache(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else if _, err := cache.NewCacheFromURI(u.String()); err != nil {
		return err
	} else {
		no.sealCache = u

		return nil
	}
}
