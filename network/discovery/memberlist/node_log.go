package memberlist

import (
	ml "github.com/hashicorp/memberlist"
	"github.com/spikeekips/mitum/util/logging"
)

func LogNode(f logging.Emitter, n *ml.Node) logging.Emitter {
	return f.Str("node_address", n.Address())
}

func LogNodeMeta(f logging.Emitter, meta NodeMeta) logging.Emitter {
	e := f.Bool("node_insecure", meta.Insecure()).Interface("meta", meta.meta)
	if meta.Publish() != nil {
		e = e.Str("node_publish", meta.Publish().String())
	}

	return e
}
