package hint

import "golang.org/x/xerrors"

func (ty Type) MarshalJSON() ([]byte, error) {
	name := ty.String()
	if len(name) < 1 {
		return nil, xerrors.Errorf("Type does not have name: type=%x", ty.Bytes())
	}

	return jsoni.Marshal(name)
}

func (ty *Type) UnmarshalJSON(b []byte) error {
	var name string
	if err := jsoni.Unmarshal(b, &name); err != nil {
		return err
	}

	t, err := TypeByName(name)
	if err != nil {
		return err
	}

	*ty = t

	return nil
}
