package cmds

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/stretchr/testify/suite"
)

type testRunNode struct {
	suite.Suite
}

func (t *testRunNode) designFile(y string) string {
	tmpfile, err := ioutil.TempFile("", "")
	t.NoError(err)

	_, err = tmpfile.Write([]byte(y))
	t.NoError(err)
	t.NoError(tmpfile.Close())

	return tmpfile.Name()
}

func (t *testRunNode) TestNew() {
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
		Run RunCommand `cmd:""`
	}{
		Run: NewRunCommand(true),
	}

	kctx, err := Context(
		[]string{
			"run",
			design,
		},
		&flags,
	)
	t.NoError(err)

	t.NoError(kctx.Run(logging.NilLogger, util.Version("v1.2.3")))
}

func TestRunNode(t *testing.T) {
	suite.Run(t, new(testRunNode))
}
