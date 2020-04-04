package util

func ConcatBytesSlice(sl ...[]byte) []byte {
	var t int
	for _, s := range sl {
		t += len(s)
	}

	n := make([]byte, t)
	var i int
	for _, s := range sl {
		i += copy(n[i:], s)
	}

	return n
}

func InStringSlice(n string, s []string) bool {
	for _, i := range s {
		if n == i {
			return true
		}
	}

	return false
}
