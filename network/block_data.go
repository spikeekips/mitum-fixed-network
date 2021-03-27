package network

import (
	"bytes"
	"io"
	"net/url"
	"path/filepath"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func FetchBlockDataThruChannel(handler BlockDataHandler, item block.BlockDataMapItem) (io.ReadCloser, error) {
	var u *url.URL
	if i, err := url.Parse(item.URL()); err != nil {
		return nil, err
	} else {
		u = i
	}

	var ro io.ReadCloser
	switch u.Scheme {
	case "file":
		i, closefunc, err := handler(u.Path)
		if closefunc != nil {
			defer func() {
				_ = closefunc()
			}()
		}

		switch {
		case err != nil:
			return nil, err
		case i == nil:
			return nil, xerrors.Errorf("empty raw block data reader returned")
		default:
			ro = i
		}
	default:
		return nil, xerrors.Errorf("%q yet supported", u.Scheme)
	}

	defer func() {
		_ = ro.Close()
	}()

	var bo io.ReadSeeker
	if b, err := io.ReadAll(ro); err != nil {
		return nil, storage.WrapFSError(err)
	} else {
		bo = bytes.NewReader(b)
	}

	// NOTE check checksum
	if i, err := util.GenerateChecksum(bo); err != nil {
		return nil, err
	} else if item.Checksum() != i {
		return nil, xerrors.Errorf("block data, %q checksum does not match; %q != %q", item.Type(), item.Checksum(), i)
	} else if _, err := bo.Seek(0, 0); err != nil {
		return nil, err
	}

	rc := util.NewNilReadCloser(bo)

	// NOTE is compressed?
	switch ext := filepath.Ext(item.URL()); ext {
	case ".gz":
		return util.NewGzipReader(rc)
	default:
		return rc, nil
	}
}
