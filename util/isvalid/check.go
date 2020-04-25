package isvalid

import "golang.org/x/xerrors"

func Check(vs []IsValider, b []byte, allowNil bool) error {
	for i, v := range vs {
		if v == nil {
			if allowNil {
				return nil
			}

			return xerrors.Errorf("%dth: nil can not be checked: type=%T", i, v)
		}
		if err := v.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}
