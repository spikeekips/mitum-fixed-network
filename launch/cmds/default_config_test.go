package cmds

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/stretchr/testify/suite"
)

type testDefaultConfig struct {
	suite.Suite
}

func (t *testDefaultConfig) TestNew() {
	flags := struct {
		DefaultConfig DefaultConfigCommand `cmd:"" name:"default_config"`
	}{
		DefaultConfig: NewDefaultConfigCommand(),
	}

	kctx, err := Context(
		[]string{
			"default_config",
		},
		&flags,
	)
	t.NoError(err)

	t.NoError(kctx.Run(logging.NilLogger, util.Version("v1.2.3")))
}

func TestDefaultConfig(t *testing.T) {
	suite.Run(t, new(testDefaultConfig))
}
