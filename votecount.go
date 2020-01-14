package mitum

import "sort"

// CheckMajority finds the majority(over threshold) set between the given sets.
// The returned value means,
// 0-N: index number of set
// -1: not yet majority
// -2: draw
func CheckMajority(total, threshold uint, set ...uint) int {
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
