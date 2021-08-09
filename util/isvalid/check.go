package isvalid

import "github.com/pkg/errors"

func Check(vs []IsValider, b []byte, allowNil bool) error {
	for i, v := range vs {
		if v == nil {
			if allowNil {
				return nil
			}

			return errors.Errorf("%dth: nil can not be checked", i)
		}
		if err := v.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}
