package isaac

import (
	"sort"
)

// FindMajority finds the majority(over threshold) set between the given sets.
// The returned value means,
// 0-N: index number of set
// -1: not yet majority
// -2: draw
func FindMajority(total, threshold uint, set ...uint) int {
	if threshold > total {
		threshold = total
	}

	if len(set) < 1 {
		return -1
	}

	var sum uint
	for _, n := range set {
		sum += n
	}

	for i, n := range set {
		if n >= total {
			return i
		}

		// check majority
		if n >= threshold {
			return i
		}
	}

	if len(set) > 0 {
		sort.Slice(
			set,
			func(i, j int) bool {
				return set[i] > set[j]
			},
		)
	}

	if total-sum+set[0] < threshold {
		return -2 // draw
	}

	return -1 // not yet
}

func FindMajorityFromSlice(total, threshold uint, s []string) (VoteResultType, string) {
	keys := map[uint]string{}
	counts := map[string]uint{}
	for _, k := range s {
		counts[k]++
	}

	set := make([]uint, len(counts))
	var i int
	for k, c := range counts {
		keys[c] = k
		set[i] = c
		i++
	}

	sort.Slice(set, func(i, j int) bool { return set[i] > set[j] })
	switch index := FindMajority(total, threshold, set...); index {
	case -1:
		return VoteResultNotYet, ""
	case -2:
		return VoteResultDraw, ""
	default:
		return VoteResultMajority, keys[set[index]]
	}
}
