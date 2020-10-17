package contestlib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

// CopyFile was derived from
// https://github.com/mactsouk/opensource.com/blob/master/cp3.go .
func CopyFile(src, dst string, bufsize int64) error {
	var sstate os.FileInfo
	if s, err := os.Stat(src); err != nil {
		return err
	} else {
		sstate = s
	}

	if !sstate.Mode().IsRegular() {
		return xerrors.Errorf("%s is not a regular file.", src)
	}

	var source *os.File
	if s, err := os.Open(filepath.Clean(src)); err != nil {
		return err
	} else {
		source = s

		defer func() {
			_ = source.Close()
		}()
	}

	if _, err := os.Stat(dst); err == nil {
		return xerrors.Errorf("File %s already exists.", dst)
	}

	var destination *os.File
	if s, err := os.Create(dst); err != nil {
		return err
	} else {
		destination = s
		defer func() {
			_ = destination.Close()
			_ = os.Chmod(dst, sstate.Mode())
		}()
	}

	buf := make([]byte, bufsize)
	for {
		if n, err := source.Read(buf); err != nil && !xerrors.Is(err, io.EOF) {
			return err
		} else if n == 0 {
			break
		} else if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}

// CopyDir is derived from
// https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	var si os.FileInfo
	switch fi, err := os.Stat(src); {
	case err != nil:
		return err
	case !fi.IsDir():
		return fmt.Errorf("source is not a directory")
	default:
		si = fi
	}

	if _, err := os.Stat(dst); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		return fmt.Errorf("destination already exists")
	}

	if err := os.MkdirAll(dst, si.Mode()); err != nil {
		return err
	} else if err := os.Chmod(dst, si.Mode()); err != nil {
		return err
	}

	var entries []os.FileInfo
	if e, err := ioutil.ReadDir(src); err != nil {
		return err
	} else {
		entries = e
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			if err := CopyFile(srcPath, dstPath, 1000); err != nil {
				return err
			}
		}
	}

	return nil
}
