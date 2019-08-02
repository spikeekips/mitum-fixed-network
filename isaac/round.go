package isaac

import "strconv"

type Round uint64

func (ro Round) String() string {
	return strconv.FormatUint(uint64(ro), 10)
}
