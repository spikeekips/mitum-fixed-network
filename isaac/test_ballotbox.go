//go:build test
// +build test

package isaac

func (bb *Ballotbox) Len() int {
	var i int
	bb.vrs.Range(func(k, _ interface{}) bool {
		i++
		return true
	})

	return i
}
