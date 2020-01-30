package isvalid

import "golang.org/x/xerrors"

func Check(vs []IsValider, b []byte) error {
	for _, v := range vs {
		if v == nil {
			return xerrors.Errorf("nil can not be checked: type=%T", v)
		}
		if err := v.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}
