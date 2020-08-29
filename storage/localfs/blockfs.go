package localfs

import (
	"io/ioutil"

	"github.com/spikeekips/mitum/storage"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func TempBlockFS(enc *jsonenc.Encoder) *storage.BlockFS {
	p, err := ioutil.TempDir("", "fs-")
	if err != nil {
		panic(err)
	}

	fs, err := NewFS(p, true)
	if err != nil {
		panic(err)
	}

	return storage.NewBlockFS(fs, enc)
}
