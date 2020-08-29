package localfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spikeekips/mitum/storage"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testLocalFS struct {
	suite.Suite
	BaseTestLocalFS
}

func (t *testLocalFS) TestNew() {
	_, err := os.Stat(t.root)
	t.NoError(err)

	{ // root is file
		fi, err := os.Create(filepath.Join(t.root, "a.txt"))
		t.NoError(err)

		_, err = NewFS(fi.Name(), false)
		t.True(xerrors.Is(err, storage.FSError))
		t.Contains(err.Error(), "is not directory")
	}

	{ // root does not exist
		_, err = NewFS(t.root+"-none", false)
		t.True(xerrors.Is(err, storage.NotFoundError))
	}

	{ // root exists
		fs, err := NewFS(t.root, false)
		t.NoError(err)
		t.Equal(t.root, fs.root)

		_ = interface{}(fs).(storage.FS)
	}
}

func (t *testLocalFS) TestFSCreate() {
	{ // root exists
		_, err := NewFS(t.root, true)
		t.NoError(err)
	}

	{ // root does not exist, IsNotExist false
		newroot := filepath.Join(t.root, "showme")
		_, err := NewFS(newroot, false)
		t.True(xerrors.Is(err, storage.NotFoundError))
	}

	{ // root does not exist, IsNotExist true
		newroot := filepath.Join(t.root, "showme")
		fs, err := NewFS(newroot, true)
		t.NoError(err)
		t.Equal(newroot, fs.root)

		_, err = os.Stat(newroot)
		t.NoError(err)
	}
}

func (t *testLocalFS) TestPath() {
	cases := []struct {
		name     string
		p        string
		expected string
		err      string
	}{
		{
			name: "invalid; empty",
			p:    "   ",
			err:  "invalid path; empty",
		},
		{
			name: "invalid; not started with /",
			p:    "showme",
			err:  "invalid path; not started with `/`",
		},
		{
			name:     "sub #0",
			p:        "/showme",
			expected: filepath.Join(t.root, "showme"),
		},
		{
			name:     "sub #1",
			p:        "/showme",
			expected: filepath.Join(t.root, "showme"),
		},
		{
			name: "rel up #0",
			p:    "../showme",
			err:  "invalid path; not started with `/`",
		},
		{
			name:     "rel up #1",
			p:        "/showme/../killme",
			expected: filepath.Join(t.root, "killme"),
		},
		{
			name: "rel up #2",
			p:    "/showme/.../killme",
			err:  "invalid path found",
		},
		{
			name:     "rel up #3",
			p:        "/showme/./killme",
			expected: filepath.Join(t.root, "showme", "killme"),
		},
	}

	fs, err := NewFS(t.root, false)
	t.NoError(err)

	for i, c := range cases {
		i := i
		c := c
		if !t.Run(
			c.name,
			func() {
				n, err := fs.insidePath(c.p)
				if err != nil {
					if len(c.err) > 0 {
						t.Contains(err.Error(), c.err, "%d: %v", i, c.name)
					} else {
						t.NoError(err, "%d: %v", i, c.name)
					}

					return
				} else if len(c.err) > 0 {
					t.NoError(xerrors.Errorf("expected error, but not occurred"), "%d: %v; expected error=%q, result=%v", i, c.name, c.err, n)

					return
				}

				t.Equal(c.expected, n, "%d: %v; %v != %v", c.expected, n)
			},
		) {
			break
		}
	}
}

func (t *testLocalFS) TestCreate() {
	fs := t.FS()

	p := "/showme"
	n, _, exists, err := fs.exists(p, false)
	t.NoError(err)
	t.False(exists)
	t.NotEmpty(n)

	err = fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	_, _, exists, err = fs.exists(p, false)
	t.NoError(err)
	t.True(exists)

	// NOTE trying to create existing file
	err = fs.Create(p, []byte("showme"), false, false)
	t.True(xerrors.Is(err, storage.FoundError))

	// NOTE trying to create existing file by force
	err = fs.Create(p, []byte("showme"), true, false)
	t.NoError(err)
}

func (t *testLocalFS) TestExists() {
	fs := t.FS()

	p := "/showme"
	exists, err := fs.Exists(p)
	t.NoError(err)
	t.False(exists)

	err = fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	exists, err = fs.Exists(p)
	t.NoError(err)
	t.True(exists)
}

func (t *testLocalFS) TestCreateButDirectory() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	exists, err := fs.Exists(p)
	t.NoError(err)
	t.True(exists)

	err = fs.Create("/showme/findme", []byte("showme"), false, false)
	t.Contains(err.Error(), "is directory")
}

func (t *testLocalFS) TestRename() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	n := "/findme/showme/killme"
	err = fs.Rename(p, n)
	t.NoError(err)

	// does not exist
	n = "/findme/showme/killme"
	err = fs.Rename(p, n)
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testLocalFS) TestRenameDirectory() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	n := "/showme/eatme"
	err = fs.Rename(p, n)
	t.NoError(err)

	found, err := fs.Exists(n)
	t.NoError(err)
	t.True(found)
}

func (t *testLocalFS) TestRenameButDirectory() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	n := "/showme/findme"
	err = fs.Rename(p, n)
	t.True(xerrors.Is(err, storage.FSError))
	t.Contains(err.Error(), "is directory")
}

func (t *testLocalFS) TestRemove() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	t.NoError(fs.Remove(p))

	// does not exist
	err = fs.Remove(p)
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testLocalFS) TestRemoveButDirectory() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	err = fs.Remove("/showme/findme")
	t.True(xerrors.Is(err, storage.FSError))
	t.Contains(err.Error(), "is directory")
}

func (t *testLocalFS) TestRemoveDirectory() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	t.NoError(fs.RemoveDirectory("/showme/findme"))

	exists, err := fs.Exists(p)
	t.NoError(err)
	t.False(exists)

	// does not exist
	err = fs.RemoveDirectory("/shome/findme")
	t.True(xerrors.Is(err, storage.NotFoundError))
}

func (t *testLocalFS) TestRemoveButFile() {
	fs := t.FS()

	p := "/showme/findme/killme"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	err = fs.RemoveDirectory(p)
	t.True(xerrors.Is(err, storage.FSError))
	t.Contains(err.Error(), "is not directory")
}

func (t *testLocalFS) TestInsidePath() {
	fs := t.FS()

	p := "/showme/findme/killme"
	n, err := fs.insidePath(p)
	t.NoError(err)

	o, err := fs.origPath(n)
	t.NoError(err)

	t.Equal(p, o)
}

func (t *testLocalFS) TestWalk() {
	fs := t.FS()

	var files []string

	p := "/a/b/c"
	err := fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)
	files = append(files, p)

	p = "/a/c"
	err = fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)
	files = append(files, p)

	p = "/c/d"
	err = fs.Create(p, []byte("showme"), false, false)
	t.NoError(err)

	var founds []string
	t.NoError(fs.Walk("/a", func(n string, fi os.FileInfo) error {
		t.False(fi.IsDir())

		founds = append(founds, n)

		return nil
	}))

	t.Equal(files, founds)
}

func TestLocalFS(t *testing.T) {
	suite.Run(t, new(testLocalFS))
}
