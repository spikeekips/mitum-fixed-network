package launcher

import "gopkg.in/yaml.v3"

func DeepCopyMap(m map[string]interface{}) (map[string]interface{}, error) {
	var n map[string]interface{}
	if b, err := yaml.Marshal(m); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &n); err != nil {
		return nil, err
	}

	return n, nil
}
