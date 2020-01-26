package isvalid

func Check(vs []IsValider, b []byte) error {
	for _, v := range vs {
		if err := v.IsValid(b); err != nil {
			return err
		}
	}

	return nil
}
