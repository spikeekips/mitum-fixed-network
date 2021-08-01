package memberlist

import (
	"net/url"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type NodeMeta struct {
	publish  *url.URL
	insecure bool
	meta     map[string]interface{}
	b        []byte
}

func NewNodeMeta(publish string, insecure bool) (NodeMeta, error) {
	u, err := network.ParseURL(publish, false)
	if err != nil {
		return NodeMeta{}, err
	}

	meta := NodeMeta{
		publish:  u,
		insecure: insecure,
	}

	meta.b, _ = util.JSON.Marshal(meta)

	return meta, meta.IsValid(nil)
}

func NewNodeMetaFromBytes(b []byte) (NodeMeta, error) {
	var meta NodeMeta
	if err := util.JSON.Unmarshal(b, &meta); err != nil {
		return NodeMeta{}, xerrors.Errorf("failed to parse NodeMeta: %w", err)
	}

	return meta, meta.IsValid(nil)
}

func (meta NodeMeta) IsValid([]byte) error {
	if len(meta.b) < 1 {
		return xerrors.Errorf("empty bytes")
	}

	return isValidPublishURL(meta.publish)
}

func (meta NodeMeta) Publish() *url.URL {
	return meta.publish
}

func (meta NodeMeta) SetPublish(u *url.URL) NodeMeta {
	meta.publish = u
	meta.b, _ = util.JSON.Marshal(meta)

	return meta
}

func (meta NodeMeta) Insecure() bool {
	return meta.insecure
}

func (meta NodeMeta) GetMeta(k string) (interface{}, bool) {
	if meta.meta == nil {
		return nil, false
	}

	v, found := meta.meta[k]

	return v, found
}

func (meta NodeMeta) AddMeta(k string, v interface{}) (NodeMeta, error) {
	if meta.meta == nil {
		meta.meta = map[string]interface{}{}
	}

	meta.meta[k] = v

	b, err := util.JSON.Marshal(meta)
	if err != nil {
		return NodeMeta{}, xerrors.Errorf("failed to marshal NodeMeta: %w", err)
	}
	meta.b = b

	return meta, nil
}

func (meta NodeMeta) Bytes() []byte {
	return meta.b
}
