package util

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

type Byter interface {
	Bytes() []byte
}

func NewByter(b []byte) Byter {
	return bytes.NewBuffer(b)
}

func CopyBytes(b []byte) []byte {
	n := make([]byte, len(b))
	copy(n, b)

	return b
}

func GenerateChecksum(i io.Reader) (string, error) {
	sha := sha256.New()
	if _, err := io.Copy(sha, i); err != nil {
		return "", xerrors.Errorf("failed to get checksum: %w", err)
	}

	return fmt.Sprintf("%x", sha.Sum(nil)), nil
}

func GenerateFileChecksum(p string) (string, error) {
	f, err := os.Open(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	return GenerateChecksum(f)
}
