package config

import (
	"net"
	"net/url"
	"strings"
	"time"

	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/ulule/limiter/v3"
)

var RateLimitHandlerMap = map[string]string{
	"seals":          quicnetwork.QuicHandlerPathGetSeals,
	"send-seal":      quicnetwork.QuicHandlerPathSendSeal,
	"blockdata-maps": quicnetwork.QuicHandlerPathGetBlockDataMaps,
	"blockdata":      quicnetwork.QuicHandlerPathGetBlockDataPattern,
	"node-info":      quicnetwork.QuicHandlerPathNodeInfo,
}

var DefaultWorldRateLimit = map[string]limiter.Rate{
	"seals":          {Period: time.Second * 10, Limit: 30},
	"send-seal":      {Period: time.Second * 10, Limit: 100},
	"blockdata-maps": {Period: time.Minute * 1, Limit: 60 * 9},
	"blockdata":      {Period: time.Minute * 1, Limit: 60 * 9},
	"node-info":      {Period: time.Second * 10, Limit: 10},
}

var DefaultSuffrageRateLimit = map[string]limiter.Rate{
	"seals":          {Period: time.Second * 10, Limit: 100},
	"send-seal":      {Period: time.Second * 10, Limit: 1000},
	"blockdata-maps": {Period: time.Second * 10, Limit: 1000},
	"blockdata":      {Period: time.Second * 10, Limit: 1000},
	"node-info":      {Period: time.Second * 10, Limit: 50},
}

var DefaultRateLimitTargetRules []RateLimitTargetRule

func init() {
	world := NewBaseRateLimitTargetRule("0.0.0.0/0", "")
	if err := world.SetRules(DefaultWorldRateLimit); err != nil {
		panic(err)
	}
	DefaultRateLimitTargetRules = []RateLimitTargetRule{world}
}

type RateLimit interface {
	Preset() map[string]RateLimitRules
	SetPreset(map[string]RateLimitRules) error
	Rules() []RateLimitTargetRule
	SetRules([]RateLimitTargetRule) error
	Cache() *url.URL
	SetCache(string) error
}

type RateLimitRules interface {
	Rules() map[string]limiter.Rate
	SetRules(map[string]limiter.Rate) error
}

type RateLimitTargetRule interface {
	RateLimitRules
	Target() string
	Preset() string
	IPNet() *net.IPNet
	SetIPNet(string) error
}

type BaseRateLimit struct {
	preset map[string]RateLimitRules
	rules  []RateLimitTargetRule
	cache  *url.URL
}

func NewBaseRateLimit(rules []RateLimitTargetRule) *BaseRateLimit {
	return &BaseRateLimit{rules: rules}
}

func (no *BaseRateLimit) Preset() map[string]RateLimitRules {
	return no.preset
}

func (no *BaseRateLimit) SetPreset(preset map[string]RateLimitRules) error {
	no.preset = preset

	return nil
}

func (no *BaseRateLimit) Rules() []RateLimitTargetRule {
	return no.rules
}

func (no *BaseRateLimit) SetRules(rules []RateLimitTargetRule) error {
	no.rules = rules

	return nil
}

func (no BaseRateLimit) Cache() *url.URL {
	return no.cache
}

func (no *BaseRateLimit) SetCache(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else if _, err := quicnetwork.RateLimitStoreFromURI(u.String()); err != nil {
		return err
	} else {
		no.cache = u

		return nil
	}
}

type BaseRateLimitRules struct {
	rules map[string]limiter.Rate
}

func NewBaseRateLimitRules(rules map[string]limiter.Rate) *BaseRateLimitRules {
	return &BaseRateLimitRules{rules: rules}
}

func (no *BaseRateLimitRules) Rules() map[string]limiter.Rate {
	return no.rules
}

func (no *BaseRateLimitRules) SetRules(rules map[string]limiter.Rate) error {
	no.rules = rules

	return nil
}

type BaseRateLimitTargetRule struct {
	*BaseRateLimitRules
	target string
	preset string
	ipnet  *net.IPNet
}

func NewBaseRateLimitTargetRule(target, preset string) *BaseRateLimitTargetRule {
	return &BaseRateLimitTargetRule{
		BaseRateLimitRules: NewBaseRateLimitRules(nil),
		target:             strings.TrimSpace(target),
		preset:             preset,
	}
}

func (no *BaseRateLimitTargetRule) Target() string {
	return no.target
}

func (no *BaseRateLimitTargetRule) Preset() string {
	return no.preset
}

func (no *BaseRateLimitTargetRule) IPNet() *net.IPNet {
	return no.ipnet
}

func (no *BaseRateLimitTargetRule) SetIPNet(s string) error {
	var target string
	if !strings.Contains(s, "/") {
		target = s + "/32"
	} else {
		target = s
	}

	if _, i, err := net.ParseCIDR(target); err != nil {
		return err
	} else {
		no.ipnet = i

		return nil
	}
}
