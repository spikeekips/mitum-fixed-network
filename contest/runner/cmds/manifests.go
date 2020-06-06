package cmds

import (
	"fmt"
	"net/url"
	"os"

	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type ManifestsCommand struct {
	URL     *url.URL `arg:"" name:"node url" help:"remote mitum url" required:""`
	Heights []int64  `arg:"" name:"height" help:"block height of manifest" required:""`
}

func (cmd *ManifestsCommand) Run(log logging.Logger) error {
	log.Debug().Interface("node_url", cmd.URL.String()).Ints64("heights", cmd.Heights).Msg("trying to get manifests")

	log.Debug().Msg("trying to get manifests thru channel")

	if i, err := requestByHeights(cmd.URL, cmd.Heights, "manifests", log); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, jsonencoder.ToString(i))
	}

	return nil
}
