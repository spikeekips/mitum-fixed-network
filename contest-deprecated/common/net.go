package common

import "github.com/spikeekips/mitum/util"

func FreePort(proto string, excludes []int) (int, error) {
	for {
		p, err := util.FreePort(proto)
		if err != nil {
			return 0, err
		}

		var found bool
		for _, e := range excludes {
			if p == e {
				found = true
				break
			}
		}
		if !found {
			return p, nil
		}
	}
}
