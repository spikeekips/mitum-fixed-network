package key // nolint

func (sp StellarPrivatekey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(sp)
}

func (sp *StellarPrivatekey) UnmarshalBSON(b []byte) error {
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		return sp.unpack(s)
	}
}

func (sp StellarPublickey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(sp)
}

func (sp *StellarPublickey) UnmarshalBSON(b []byte) error {
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		return sp.unpack(s)
	}
}
