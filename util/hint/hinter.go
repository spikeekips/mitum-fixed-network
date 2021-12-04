package hint

type BaseHinter struct {
	ht Hint
}

func NewBaseHinter(ht Hint) BaseHinter {
	return BaseHinter{ht: ht}
}

func (ht BaseHinter) Hint() Hint {
	return ht.ht
}

func (ht BaseHinter) SetHint(n Hint) Hinter {
	ht.ht = n

	return ht
}

func (ht BaseHinter) IsValid([]byte) error {
	return ht.ht.IsValid(nil)
}
