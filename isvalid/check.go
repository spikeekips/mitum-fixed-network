package isvalid

import "golang.org/x/xerrors"

func Check(vs []IsValider, b []byte, allowNil bool) error {
	for _, v := range vs {
		if v == nil {
			if allowNil {
				return nil
			}

			return xerrors.Errorf("nil can not be checked: type=%T", v)
		}
		if err := v.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}
