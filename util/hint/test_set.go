package hint

func (hs *Hintset) TestAdd(ht Hinter) error {
	if err := hs.Add(ht); err != nil {
		panic(err)
	}

	return nil
}
