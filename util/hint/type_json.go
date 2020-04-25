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
		return nil, xerrors.Errorf("Type does not have name: %s", ty.Verbose())
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

	return ty.unpack(o.C, o.N)
}
