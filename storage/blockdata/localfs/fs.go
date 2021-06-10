package localfs

import (
	"io/fs"
	"os"
	"path/filepath"
)

type FS struct {
	root string
}

func NewFS(root string) FS {
	return FS{root: root}
}

func (f FS) Open(p string) (fs.File, error) {
	return os.Open(filepath.Clean(filepath.Join(f.root, p)))
}
