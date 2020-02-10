package hint

import (
	"encoding/hex"

	"golang.org/x/xerrors"
)

type typeJSON struct {
	N string `json:"name"`
	C string `json:"code"`
}

func (ty Type) MarshalJSON() ([]byte, error) {
	name := ty.String()
	if len(name) < 1 {
		return nil, xerrors.Errorf("Type does not have name: %v", ty.Verbose())
	}

	return jsoni.Marshal(typeJSON{
		N: name,
		C: hex.EncodeToString(ty.Bytes()),
	})
}

func (ty *Type) UnmarshalJSON(b []byte) error {
	var o typeJSON
	if err := jsoni.Unmarshal(b, &o); err != nil {
		return err
	}

	var n [2]byte
	if d, err := hex.DecodeString(o.C); err != nil {
		return err
	} else {
		copy(n[:], d)
	}

	nt := Type(n)

	if t, err := TypeByName(o.N); err != nil {
		return err
	} else if !nt.Equal(t) {
		return NewTypeDoesNotMatchError(t, nt)
	}

	*ty = nt

	return nil
}
