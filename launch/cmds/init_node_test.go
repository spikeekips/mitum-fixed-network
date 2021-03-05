package cmds

import (
	"os"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/stretchr/testify/suite"
)

type testInitNode struct {
	suite.Suite
}

func (t *testInitNode) designFile(y string) string {
	tmpfile, err := os.CreateTemp("", "")
	t.NoError(err)

	_, err = tmpfile.Write([]byte(y))
	t.NoError(err)
	t.NoError(tmpfile.Close())

	return tmpfile.Name()
}

func (t *testInitNode) TestNew() {
	y := `
network-id: show me
address: node-010a:0.0.1
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef-0112:0.0.1
storage:
  uri: mongodb://localhost:27017/@@db@@
  blockfs:
    path: /tmp/@@db@@
`

	y = strings.ReplaceAll(y, "@@db@@", util.UUID().String())
	design := t.designFile(y)
	defer os.Remove(design)

	flags := struct {
		Init InitCommand `cmd:""`
	}{
		Init: NewInitCommand(true),
	}

	kctx, err := Context(
		[]string{
			"init",
			design,
		},
		&flags,
	)
	t.NoError(err)

	t.NoError(kctx.Run(logging.NilLogger, util.Version("v1.2.3")))
}

func TestInitNode(t *testing.T) {
	suite.Run(t, new(testInitNode))
}
