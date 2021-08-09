package memberlist

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

func (meta NodeMeta) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	for k := range meta.meta {
		m[k] = meta.meta[k]
	}

	if meta.publish != nil {
		m["p"] = meta.publish.String()
	}

	m["i"] = meta.insecure

	return util.JSON.Marshal(m)
}

func (meta *NodeMeta) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := util.JSON.Unmarshal(b, &m); err != nil {
		return err
	}

	meta.meta = map[string]interface{}{}
	for k := range m {
		switch k {
		case "p":
			i, ok := m["p"].(string)
			if !ok {
				return errors.Errorf("invalid publish, %T", m["p"])
			}

			if len(i) > 0 {
				u, err := network.ParseURL(i, false)
				if err != nil {
					return err
				}
				meta.publish = u
			}

			continue
		case "i":
			meta.insecure = m["i"].(bool)
			continue
		}

		meta.meta[k] = m[k]
	}

	meta.b = b

	return nil
}
