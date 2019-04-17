package common

type Uint uint64

func (i Uint) AddOK(n Uint) (Uint, bool) {
	c := i + n
	if (c > i) == (n > 0) {
		return c, true
	}
	return 0, false
}

func (i Uint) SubOK(n Uint) (Uint, bool) {
	if i < n {
		return 0, false
	}

	return i - n, true
}

func (i Uint) MulOK(n Uint) (Uint, bool) {
	if i == 0 || n == 0 {
		return 0, true
	}

	if n == 1 {
		return i, true
	}

	c := i * n
	if c/n == i {
		return c, true
	}

	return 0, false
}

func (i Uint) DivOK(n Uint) (Uint, bool) {
	if n == 0 {
		return 0, false
	} else if i == 0 {
		return 0, true
	}

	if n == 1 {
		return i, true
	}

	return i / n, true
}
