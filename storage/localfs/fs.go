package localfs

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spikeekips/mitum/storage"
	"golang.org/x/xerrors"
)

var (
	DefaultFilePermission      os.FileMode = 0o640
	DefaultDirectoryPermission os.FileMode = 0o750
	osOpenFile                             = os.OpenFile
	// TODO At this time, Sun 30 Aug 2020 05:44:44 AM KST, golangci-lint
	// produced 'G304: Potential file inclusion via variable (gosec)', but
	// filepath.Clean already applied.
)

func mkdirAll(p string, perm os.FileMode) error {
	if err := os.MkdirAll(p, perm); err != nil {
		return err
	} else if err := os.Chmod(p, perm); err != nil {
		return err
	} else {
		return nil
	}
}

func openFile(p string, flag int, perm os.FileMode) (*os.File, error) {
	if f, err := osOpenFile(filepath.Clean(p), flag, perm); err != nil {
		return nil, err
	} else if err := f.Chmod(perm); err != nil {
		return nil, err
	} else {
		return f, nil
	}
}

type FS struct {
	sync.RWMutex
	root string
}

func NewFS(root string, ifNotCreate bool) (*FS, error) {
	if fi, err := os.Stat(root); err != nil {
		if !os.IsNotExist(err) {
			return nil, storage.WrapFSError(err)
		} else if !ifNotCreate {
			return nil, storage.WrapFSError(err)
		}

		if err := mkdirAll(root, DefaultDirectoryPermission); err != nil {
			return nil, storage.WrapFSError(err)
		}
	} else if !fi.IsDir() {
		return nil, storage.FSError.Errorf("root, %q is not directory", root)
	}

	// NOTE check writable
	if p, err := ioutil.TempDir(root, ".temp"); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		_ = os.RemoveAll(p)
	}

	if a, err := filepath.Abs(root); err != nil {
		return nil, err
	} else {
		return &FS{root: a}, nil
	}
}

func (fs *FS) Root() string {
	return fs.root
}

func (fs *FS) Clean(remove bool) error {
	fs.Lock()
	defer fs.Unlock()

	if remove {
		if err := os.RemoveAll(fs.root); err != nil {
			return storage.WrapFSError(err)
		}

		return nil
	}

	if files, err := ioutil.ReadDir(fs.root); err != nil {
		return storage.WrapFSError(err)
	} else {
		for _, f := range files {
			if err := os.RemoveAll(filepath.Join(fs.root, f.Name())); err != nil {
				return storage.WrapFSError(err)
			}
		}
	}

	return nil
}

func (fs *FS) Exists(p string) (bool, error) {
	fs.RLock()
	defer fs.RUnlock()

	_, _, exists, err := fs.exists(p, false)

	return exists, err
}

func (fs *FS) Open(p string) (io.ReadCloser, error) {
	fs.RLock()
	defer fs.RUnlock()

	if n, fi, exists, err := fs.exists(p, false); err != nil {
		return nil, err
	} else if !exists {
		return nil, storage.NotFoundError.Errorf("%q does not exist", p)
	} else if fi.IsDir() {
		return nil, storage.FSError.Errorf("%q is directory", p)
	} else if f, err := os.Open(filepath.Clean(n)); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		return f, nil
	}
}

func (fs *FS) Create(p string, b []byte, force bool, compress bool) error {
	fs.RLock()
	defer fs.RUnlock()

	var f *os.File

	var n string
	switch i, _, exists, err := fs.exists(p, false); {
	case err != nil:
		return err
	case exists:
		if !force {
			return storage.FoundError.Errorf("already exists")
		}

		if err := os.RemoveAll(i); err != nil {
			return storage.WrapFSError(err)
		}
		n = i
	default:
		n = i
	}

	switch i, _, exists, err := fs.exists(filepath.Dir(p), true); {
	case err != nil:
		return err
	case !exists:
		if err := mkdirAll(i, DefaultDirectoryPermission); err != nil {
			return storage.WrapFSError(err)
		}
	}

	if i, err := openFile(n, os.O_CREATE|os.O_WRONLY, DefaultFilePermission); err != nil {
		return storage.WrapFSError(err)
	} else {
		f = i
	}

	defer func() {
		_ = f.Close()
	}()

	if compress {
		gw := gzip.NewWriter(f)
		defer func() {
			_ = gw.Close()
		}()

		if _, err := gw.Write(b); err != nil {
			return storage.WrapFSError(err)
		} else {
			return nil
		}
	} else {
		if _, err := f.Write(b); err != nil {
			return storage.WrapFSError(err)
		} else {
			return nil
		}
	}
}

func (fs *FS) CreateDirectory(p string) error {
	if ns, err := fs.insidePath(p); err != nil {
		return err
	} else if err := mkdirAll(ns, DefaultDirectoryPermission); err != nil {
		return storage.WrapFSError(err)
	}

	return nil
}

func (fs *FS) Rename(s string, t string) error {
	fs.RLock()
	defer fs.RUnlock()

	var isDir bool
	if ns, err := fs.insidePath(s); err != nil {
		return err
	} else if fi, err := os.Stat(ns); err != nil {
		return storage.WrapFSError(err)
	} else {
		isDir = fi.IsDir()
	}

	var ns, nt string
	switch n, _, exists, err := fs.exists(s, isDir); {
	case err != nil:
		return err
	case !exists:
		return storage.NotFoundError.Errorf("%q does not exist", s)
	default:
		ns = n
	}

	switch n, _, exists, err := fs.exists(t, isDir); {
	case err != nil:
		return err
	case exists:
		return storage.FoundError.Errorf("%q exists", t)
	default:
		switch d, _, exists, err := fs.exists(filepath.Dir(t), true); {
		case err != nil:
			return err
		case !exists:
			if err := mkdirAll(d, DefaultDirectoryPermission); err != nil {
				return storage.WrapFSError(err)
			}
		}
		nt = n
	}

	if err := os.Rename(ns, nt); err != nil {
		return storage.WrapFSError(err)
	} else {
		return nil
	}
}

func (fs *FS) Remove(p string) error {
	fs.RLock()
	defer fs.RUnlock()

	if n, _, exists, err := fs.exists(p, false); err != nil {
		return err
	} else if !exists {
		return storage.NotFoundError.Errorf("%q does not exist", p)
	} else if err := os.Remove(n); err != nil {
		return storage.WrapFSError(err)
	} else {
		return nil
	}
}

func (fs *FS) RemoveDirectory(d string) error {
	fs.RLock()
	defer fs.RUnlock()

	if n, _, exists, err := fs.exists(d, true); err != nil {
		return err
	} else if !exists {
		return storage.NotFoundError.Errorf("%q does not exist", d)
	} else if err := os.RemoveAll(n); err != nil {
		return storage.WrapFSError(err)
	} else {
		return nil
	}
}

func (fs *FS) Stat(p string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()

	if n, err := fs.insidePath(p); err != nil {
		return nil, err
	} else if fi, err := os.Stat(n); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		return fi, nil
	}
}

func (fs *FS) Walk(p string, f storage.WalkFunc) error {
	fs.RLock()
	defer fs.RUnlock()

	if n, _, exists, err := fs.exists(p, true); err != nil {
		return err
	} else if !exists {
		return storage.NotFoundError.Errorf("%q does not exist", p)
	} else if err := filepath.Walk(n, func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			return storage.WrapFSError(err)
		} else if fi.IsDir() {
			return nil
		}

		if o, err := fs.origPath(fp); err != nil {
			return err
		} else {
			return f(o, storage.NewEmptyFileInfo(o, false))
		}
	}); err != nil {
		return storage.WrapFSError(err)
	} else {
		return nil
	}
}

func (fs *FS) exists(p string, isDir bool) (string, os.FileInfo, bool, error) {
	if n, err := fs.insidePath(p); err != nil {
		return "", nil, false, err
	} else if fi, err := os.Stat(n); err != nil {
		if os.IsNotExist(err) {
			return n, storage.NewEmptyFileInfo(n, isDir), false, nil
		} else {
			return n, nil, false, storage.WrapFSError(err)
		}
	} else {
		if isDir && !fi.IsDir() {
			return n, nil, false, storage.FSError.Errorf("%q is not directory", p)
		} else if !isDir && fi.IsDir() {
			return n, nil, false, storage.FSError.Errorf("%q is directory", p)
		}

		return n, fi, true, nil
	}
}

func (fs *FS) insidePath(p string) (string, error) {
	k := strings.TrimSpace(p)
	if len(k) < 1 {
		return "", xerrors.Errorf("invalid path; empty")
	} else if !strings.HasPrefix(k, "/") {
		return "", xerrors.Errorf("invalid path; not started with `/`")
	}

	n := filepath.Join(fs.root, k)
	if !strings.HasPrefix(n, fs.root) {
		return "", storage.FSError.Errorf("p, %q is out of root, %q", p, fs.root)
	} else if strings.Contains(n, "..") {
		return "", storage.FSError.Errorf("invalid path found, %q", n)
	}

	return n, nil
}

func (fs *FS) origPath(p string) (string, error) {
	if !strings.HasPrefix(p, fs.root) {
		return "", storage.FSError.Errorf("p, %q is out of root, %q", p, fs.root)
	} else {
		return p[len(fs.root):], nil
	}
}
