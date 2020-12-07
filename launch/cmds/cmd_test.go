package cmds

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/suite"
)

type testMain struct {
	suite.Suite
}

func (t *testMain) TestNew() {
	var flags struct {
		A string
		B int
	}

	kctx, err := Context([]string{"--a", "showme", "--b", "3"}, &flags)
	t.NoError(err)

	t.Equal(DefaultName, kctx.Model.Name)

	t.Equal("showme", flags.A)
	t.Equal(3, flags.B)
}

func (t *testMain) TestOverrideName() {
	var flags struct {
		A string
		B int
	}

	kctx, err := Context([]string{"--a", "showme", "--b", "3"}, &flags, kong.Name("find-me"))
	t.NoError(err)

	t.Equal("find-me", kctx.Model.Name)
}

func (t *testMain) TestOverrideVars() {
	var flags struct {
		A string `help:"default: ${kill}"`
		B int
	}

	kctx, err := Context([]string{"--a", "showme", "--b", "3"}, &flags, kong.Vars{
		"kill": "me",
	})
	t.NoError(err)

	var out bytes.Buffer
	kctx.Stdout = &out
	kctx.PrintUsage(false)

	t.Contains(out.String(), "default: me")
}

func TestMain(t *testing.T) {
	suite.Run(t, new(testMain))
}
