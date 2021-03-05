package cmds

import (
	"bytes"
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

type FileLoad []byte

func (v FileLoad) MarshalText() ([]byte, error) {
	return []byte(v), nil
}

func (v *FileLoad) UnmarshalText(b []byte) error {
	var body []byte
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := LoadFromStdInput(); err != nil {
			return err
		} else {
			body = c
		}
	} else if c, err := os.ReadFile(filepath.Clean(string(b))); err != nil {
		return err
	} else {
		body = c
	}

	if len(body) < 1 {
		return xerrors.Errorf("empty file")
	}

	*v = body

	return nil
}

func (v FileLoad) Bytes() []byte {
	return []byte(v)
}

func (v FileLoad) String() string {
	return string(v)
}
