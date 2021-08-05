package memberlist

import (
	"github.com/rs/zerolog"
)

func (meta NodeMeta) MarshalZerologObject(e *zerolog.Event) {
	e = e.Bool("node_insecure", meta.Insecure()).Interface("meta", meta.meta)

	if meta.Publish() != nil {
		e.Stringer("node_publish", meta.Publish())
	}
}
