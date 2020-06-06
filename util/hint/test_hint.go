package hint

import "github.com/spikeekips/mitum/util"

func MustHintWithType(t Type, version util.Version, name string) Hint {
	if err := registerType(t, name); err != nil {
		panic(err)
	}

	ht := Hint{t: t, version: version}
	if err := ht.IsValid(nil); err != nil {
		panic(err)
	}

	return ht
}
