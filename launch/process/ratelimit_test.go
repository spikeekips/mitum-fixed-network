package process

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/ulule/limiter/v3"
)

func (rl *RateLimit) RateByString(s string) limiter.Rate {
	ip := net.ParseIP(s)
	if ip == nil {
		return rl.defaultRate
	}

	return rl.Rate(ip)
}

type testRateLimit struct {
	suite.Suite
}

func (t *testRateLimit) rule(cidr string, limit int64, period time.Duration) RateLimitRule {
	_, ipnet, err := net.ParseCIDR(cidr)
	t.NoError(err)

	return NewRateLimiterRule(ipnet, limiter.Rate{Limit: limit, Period: period})
}

func (t *testRateLimit) TestNew() {
	rules := []RateLimitRule{
		t.rule("192.168.1.1/32", 1, time.Second),
		t.rule("192.168.1.2/32", 2, time.Second*2),
	}
	d := limiter.Rate{Limit: 3, Period: time.Second * 3}

	rl := NewRateLimit(rules, d)

	var r limiter.Rate
	r = rl.RateByString("192.168.1.1")
	t.Equal(int64(1), r.Limit)
	t.Equal(time.Second, r.Period)

	r = rl.RateByString("192.168.1.2")
	t.Equal(int64(2), r.Limit)
	t.Equal(time.Second*2, r.Period)

	r = rl.RateByString("192.168.1.100")
	t.Equal(int64(3), r.Limit)
	t.Equal(time.Second*3, r.Period)

	r = rl.RateByString("192.168.1.100")
	t.Equal(int64(3), r.Limit)
	t.Equal(time.Second*3, r.Period)
}

func (t *testRateLimit) TestMatch() {
	rules := []RateLimitRule{
		t.rule("192.168.1.3/32", 1, time.Second),
		t.rule("192.168.1.2/24", 2, time.Second*2),
		t.rule("192.168.1.0/16", 3, time.Second*3),
	}
	d := limiter.Rate{Limit: 55, Period: time.Second * 55}

	rl := NewRateLimit(rules, d)

	cases := []struct {
		name     string
		ip       string
		expected string
	}{
		{"3/32", "192.168.1.3", "1/1s"},
		{"2/24 #0", "192.168.1.9", "2/2s"},
		{"2/24 #1", "192.168.1.100", "2/2s"},
		{"0/16 #0", "192.168.2.9", "3/3s"},
		{"0/16 #1", "192.168.9.9", "3/3s"},
		{"default #0", "192.167.9.9", "55/55s"},
		{"default #1", "127.0.0.9", "55/55s"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				r := rl.RateByString(c.ip)
				result := fmt.Sprintf("%d/%s", r.Limit, r.Period.String())

				t.Equal(c.expected, result, "%d: %v; %v != %v", i, c.name, c.expected, result)
			},
		)
	}
}

func TestRateLimit(t *testing.T) {
	suite.Run(t, new(testRateLimit))
}
