package isvalid

func Check(b []byte, allowNil bool, vs ...IsValider) error {
	for i, v := range vs {
		if v == nil {
			if allowNil {
				return nil
			}

			return InvalidError.Errorf("%dth: nil can not be checked", i)
		}
		if err := v.IsValid(b); err != nil {
			return InvalidError.Wrap(err)
		}
	}

	return nil
}

func CheckFunc(fs []func() error) error {
	for i := range fs {
		if fs[i] == nil {
			return InvalidError.Errorf("%dth: nil func", i)
		}

		if err := fs[i](); err != nil {
			return InvalidError.Wrap(err)
		}
	}

	return nil
}
