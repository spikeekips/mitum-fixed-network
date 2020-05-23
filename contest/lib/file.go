package contestlib

import (
	"io"
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
		if n, err := source.Read(buf); err != nil && err != io.EOF {
			return err
		} else if n == 0 {
			break
		} else if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}
