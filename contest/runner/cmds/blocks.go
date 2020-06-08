package cmds

import (
	"fmt"
	"net/url"
	"os"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type BlocksCommand struct {
	URL     *url.URL `arg:"" name:"node url" help:"remote mitum url" required:""`
	Heights []int64  `arg:"" name:"height" help:"block height of blocks" required:""`
}

func (cmd *BlocksCommand) Run(log logging.Logger) error {
	log.Debug().Interface("node_url", cmd.URL.String()).Ints64("heights", cmd.Heights).Msg("trying to get blocks")

	log.Debug().Msg("trying to get blocks thru channel")

	if i, err := requestByHeights(cmd.URL, cmd.Heights, "blocks", log); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(i))
	}

	return nil
}
