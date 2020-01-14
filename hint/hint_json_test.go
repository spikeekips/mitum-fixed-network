package hint

func (t *testHint) TestMarshalJSON() {
	ty := Type([2]byte{0xff, 0xf0})
	v := "0.1"

	_ = RegisterType(ty, "0xfff0")

	h, err := NewHint(ty, v)
	t.NoError(err)

	b, err := jsoni.Marshal(h)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsoni.Unmarshal(b, &m))

	t.Equal(h.Type().String(), m["type"])
	t.Equal(h.Version(), m["version"])

	// unmarshal
	var uh Hint
	t.NoError(jsoni.Unmarshal(b, &uh))
	t.Equal(h.Type(), uh.Type())
	t.Equal(h.Version(), uh.Version())
}
