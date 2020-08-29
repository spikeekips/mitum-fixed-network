package storage

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

type WalkFunc func(fp string, fi os.FileInfo) error

type FS interface {
	Root() string
	Open(string) (io.ReadCloser, error)
	Exists(string) (bool, error)
	Create(string, []byte, bool, bool) error
	CreateDirectory(string) error
	Rename(string, string) error
	Remove(string) error
	RemoveDirectory(string) error
	Stat(string) (os.FileInfo, error)
	Walk(string, WalkFunc) error
	Clean(bool) error
}

type EmptyFileInfo struct {
	p     string
	isDir bool
}

func NewEmptyFileInfo(p string, isDir bool) EmptyFileInfo {
	return EmptyFileInfo{p: filepath.Base(p), isDir: isDir}
}

func (df EmptyFileInfo) Name() string {
	return df.p
}

func (df EmptyFileInfo) Size() int64 {
	return 0
}

func (df EmptyFileInfo) Mode() os.FileMode {
	return os.ModeIrregular
}

func (df EmptyFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (df EmptyFileInfo) IsDir() bool {
	return df.isDir
}

func (df EmptyFileInfo) Sys() interface{} {
	return nil
}
