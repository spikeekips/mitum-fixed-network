package network

import (
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type testURL struct {
	suite.Suite
}

func (t *testURL) TestNormalizeURL() {
	cases := []struct {
		name     string
		s        string
		expected string
	}{
		{name: "nil url", s: "", expected: ""},
		{name: "full", s: "https://findme:334/show/me?a=b#f", expected: "https://findme:334/show/me?a=b#f"},
		{name: "full, empty https port", s: "https://findme/show/me?a=b#f", expected: "https://findme:443/show/me?a=b#f"},
		{name: "full, empty http port", s: "http://findme/show/me?a=b#f", expected: "http://findme:80/show/me?a=b#f"},
		{name: "full, empty unknown port", s: "what://findme/show/me?a=b#f", expected: "what://findme:0/show/me?a=b#f"},
		{name: "/ path", s: "https://findme/", expected: "https://findme:443"},
		{name: "blank path", s: "https://findme", expected: "https://findme:443"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				var u *url.URL
				if len(c.s) > 1 {
					j, err := url.Parse(c.s)
					if err != nil {
						panic(err)
					}
					u = j
				}

				j := NormalizeURL(u)
				if c.expected == "" {
					t.Nil(j, "%d: %v", i, c.name)
				} else {
					t.Equal(c.expected, j.String(), "%d: %v", i, c.name)
				}
			},
		)
	}
}

func (t *testURL) TestIsValidURL() {
	cases := []struct {
		name string
		s    string
		err  string
	}{
		{name: "nil url", s: "", err: "empty url"},
		{name: "empty port", s: "http://local"},
		{name: "with port", s: "http://local:334"},
		{name: "with path", s: "http://local:334/showme"},
		{name: "empty scheme", s: "//findme:334", err: "empty scheme"},
		{name: "empty host", s: "http://:334", err: "empty host"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				var u *url.URL
				if len(c.s) > 1 {
					j, err := url.Parse(c.s)
					if err != nil {
						panic(err)
					}
					u = j
				}

				err := IsValidURL(u)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(errors.Errorf("expected %q, but nil error", c.err), "%d: %v", i, c.name)
					} else {
						t.Contains(err.Error(), c.err, "%d: %v", i, c.name)
					}

					return
				} else if err != nil {
					t.NoError(errors.Errorf("expected nil error, but %+v", err), "%d: %v", i, c.name)

					return
				}
			},
		)
	}
}

func (t *testURL) TestParseCombinedNodeURL() {
	cases := []struct {
		name     string
		s        string
		expected string
		insecure bool
		err      string
	}{
		{name: "nil url", s: "", err: "empty url"},
		{name: "invalid url", s: "http://:334", err: "empty host"},
		{name: "insecure", s: "http://showme:334#insecure", expected: "http://showme:334", insecure: true},
		{name: "insecure, empty port", s: "http://showme#insecure", expected: "http://showme:80", insecure: true},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				var u *url.URL
				if len(c.s) > 1 {
					j, err := url.Parse(c.s)
					if err != nil {
						panic(err)
					}
					u = j
				}

				ru, insecure, err := ParseCombinedNodeURL(u)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(errors.Errorf("expected %q, but nil error", c.err), "%d: %v", i, c.name)

						return
					}

					t.Contains(err.Error(), c.err, "%d: %v", i, c.name)

					return
				} else if err != nil {
					t.NoError(errors.Errorf("expected nil error, but %+v", err), "%d: %v", i, c.name)

					return
				}

				t.Equal(c.expected, ru.String(), "%d: %v", i, c.name)
				t.Equal(c.insecure, insecure, "%d: %v", i, c.name)
			},
		)
	}
}

func TestURL(t *testing.T) {
	suite.Run(t, new(testURL))
}
