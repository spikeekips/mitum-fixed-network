package localfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"golang.org/x/xerrors"
)

var (
	LocalFSBlockDataType = hint.Type("localfs-blockdata")
	LocalFSBlockDataHint = hint.NewHint(LocalFSBlockDataType, "v0.0.1")
)

var (
	BlockDirectoryHeightFormat     = "%021s"
	BlockDirectoryRemovedTouchFile = ".removed"
	BlockFileFormats               = "%d-%s-%s.jsonld.gz" // <height>-<data>-<checksum>.jsonld.gz
	BlockFileGlobFormats           = "%d-%s-*.jsonld.gz"  // <height>-<data>-*.jsonld.gz
)

type BlockData struct {
	sync.RWMutex
	root    string
	encoder *jsonenc.Encoder
	writer  blockdata.Writer
	fs      FS
}

func NewBlockData(root string, encoder *jsonenc.Encoder) *BlockData {
	return &BlockData{
		root:    root,
		encoder: encoder,
		writer:  blockdata.NewDefaultWriter(encoder),
		fs:      NewFS(root),
	}
}

func (st *BlockData) Initialize() error {
	i, err := filepath.Abs(st.root)
	if err != nil {
		return storage.WrapFSError(err)
	}
	st.root = i

	if fi, err := os.Stat(st.root); err != nil {
		return storage.WrapFSError(err)
	} else if !fi.IsDir() {
		return storage.FSError.Errorf("storage root, %q is not directory", st.root)
	}

	return nil
}

func (*BlockData) Hint() hint.Hint {
	return LocalFSBlockDataHint
}

func (*BlockData) IsLocal() bool {
	return true
}

func (st *BlockData) Writer() blockdata.Writer {
	return st.writer
}

func (st *BlockData) Exists(height base.Height) (bool, error) {
	st.RLock()
	defer st.RUnlock()

	switch found, removed, err := st.exists(height); {
	case err != nil:
		return found, err
	case removed:
		return false, nil
	default:
		return found, nil
	}
}

func (st *BlockData) ExistsReal(height base.Height) (exists bool, removed bool, err error) {
	st.RLock()
	defer st.RUnlock()

	return st.exists(height)
}

// Remove removes block directory by height. Remove does not remove the
// directory and inside files, it just creates .removd file with time. .removed
// file helps CleanByHeight to clean directories.
func (st *BlockData) Remove(height base.Height) error {
	st.Lock()
	defer st.Unlock()

	return st.remove(height)
}

func (st *BlockData) remove(height base.Height) error {
	switch found, removed, err := st.exists(height); {
	case err != nil:
		return err
	case removed:
		return nil
	case !found:
		return util.NotFoundError.Errorf("block directory not found, %v", height)
	}

	removedFile := []byte(localtime.RFC3339(localtime.UTCNow()))

	d := st.heightDirectory(height, true)
	if i, err := os.OpenFile(
		filepath.Clean(filepath.Join(d, BlockDirectoryRemovedTouchFile)),
		os.O_CREATE|os.O_WRONLY,
		DefaultFilePermission,
	); err != nil {
		return storage.WrapFSError(err)
	} else if _, err := i.Write(removedFile); err != nil {
		return storage.WrapFSError(err)
	} else {
		return nil
	}
}

// RemoveAll removes directory and it's inside files.
func (st *BlockData) RemoveAll(height base.Height) error {
	st.Lock()
	defer st.Unlock()

	switch found, _, err := st.exists(height); {
	case err != nil:
		return err
	case !found:
		return util.NotFoundError.Errorf("block directory not found, %v", height)
	}

	if err := os.RemoveAll(st.heightDirectory(height, true)); err != nil {
		return storage.WrapFSError(err)
	}

	return nil
}

func (st *BlockData) Clean(remove bool) error {
	st.Lock()
	defer st.Unlock()

	return st.clean(remove)
}

func (st *BlockData) NewSession(height base.Height) (blockdata.Session, error) {
	st.Lock()
	defer st.Unlock()

	i, err := os.MkdirTemp(st.root, ".session")
	if err != nil {
		return nil, err
	}
	return NewSession(i, st.writer, height)
}

func (st *BlockData) SaveSession(session blockdata.Session) (block.BlockDataMap, error) {
	st.Lock()
	defer st.Unlock()

	ss, ok := session.(*Session)
	if !ok {
		return nil, xerrors.Errorf("only localfs.Session only allowed for localfs blockdata, not %T", session)
	}

	mapData, err := ss.done()
	if err != nil {
		return nil, err
	}

	// NOTE move items to none-temporary place
	b := st.heightDirectory(ss.Height(), false)
	if err = st.createDirectory(filepath.Join(st.root, b)); err != nil {
		return nil, err
	}

	newMapData, err := st.moveItemFiles(b, mapData)
	if err != nil {
		return nil, err
	}
	_ = ss.clean()

	i, err := newMapData.UpdateHash()
	if err != nil {
		return nil, err
	}
	newMapData = i

	if err := newMapData.IsValid(nil); err != nil {
		return nil, err
	} else if err := newMapData.Exists(st.root); err != nil {
		return nil, err
	}

	return newMapData, nil
}

func (st *BlockData) FS() fs.FS {
	return st.fs
}

func (st *BlockData) Root() string {
	return st.root
}

func (st *BlockData) exists(height base.Height) (exists bool, removed bool, err error) {
	d := st.heightDirectory(height, true)
	if fi, err := os.Stat(d); err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}

		return false, false, storage.WrapFSError(err)
	} else if !fi.IsDir() {
		return true, false, storage.FSError.Errorf("block directory, %q is not directory, %q", d, fi.Mode().String())
	}

	// NOTE check removed file
	switch _, err := os.Stat(filepath.Join(d, BlockDirectoryRemovedTouchFile)); {
	case err == nil:
		return true, true, nil
	case !os.IsNotExist(err):
		return true, false, err
	default:
		return true, false, nil
	}
}

func (st *BlockData) clean(remove bool) error {
	if remove {
		if err := os.RemoveAll(st.root); err != nil {
			return storage.WrapFSError(err)
		}

		return nil
	}

	files, err := os.ReadDir(st.root)
	if err != nil {
		return storage.WrapFSError(err)
	}
	for _, f := range files {
		if err := os.RemoveAll(filepath.Join(st.root, f.Name())); err != nil {
			return storage.WrapFSError(err)
		}
	}

	return nil
}

func (*BlockData) createDirectory(p string) error {
	if _, err := os.Stat(p); err != nil {
		if !os.IsNotExist(err) {
			return storage.WrapFSError(err)
		}
	} else {
		if err := os.RemoveAll(p); err != nil {
			return storage.WrapFSError(err)
		}
	}

	if err := os.MkdirAll(p, DefaultDirectoryPermission); err != nil {
		return err
	} else if err := os.Chmod(p, DefaultDirectoryPermission); err != nil {
		return err
	} else {
		return nil
	}
}

func (st *BlockData) heightDirectory(height base.Height, abs bool) string {
	b := HeightDirectory(height)
	if !abs {
		return b
	}
	return filepath.Join(st.root, b)
}

func (st *BlockData) moveItemFiles(b string, mapData block.BaseBlockDataMap) (block.BaseBlockDataMap, error) {
	nm := mapData
	oldDirs := map[string]struct{}{}
	for dataType := range mapData.Items() {
		item := mapData.Items()[dataType]

		oldDirs[filepath.Dir(item.URLBody())] = struct{}{}
		newPath := filepath.Join(b, filepath.Base(item.URLBody()))

		if err := os.Rename(item.URLBody(), filepath.Join(st.root, newPath)); err != nil {
			return nm, err
		}

		i, err := nm.SetItem(item.SetFile(newPath))
		if err != nil {
			return nm, err
		}
		nm = i
	}

	return nm, nil
}
