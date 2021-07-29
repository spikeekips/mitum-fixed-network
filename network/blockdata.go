package network

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func FetchBlockDataThruChannel(handler BlockDataHandler, item block.BlockDataMapItem) (io.ReadCloser, error) {
	u, err := url.Parse(item.URL())
	if err != nil {
		return nil, err
	}

	var ro io.Reader
	switch u.Scheme {
	case "file":
		i, closefunc, e := handler(u.Path)
		if closefunc != nil {
			defer func() {
				_ = closefunc()
			}()
		}

		switch {
		case e != nil:
			return nil, e
		case i == nil:
			return nil, xerrors.Errorf("empty raw block data reader returned")
		default:
			ro = i
		}
	default:
		return nil, xerrors.Errorf("%q not yet supported", u.Scheme)
	}

	b, err := io.ReadAll(ro)
	if err != nil {
		return nil, storage.WrapFSError(err)
	}
	bo := bytes.NewReader(b)

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

func FetchBlockDataFromRemote(ctx context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	u, err := url.Parse(item.URL())
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "file":
		return nil, xerrors.Errorf("%q is not remote", u.String())
	case "http", "https": // nolint:goconst
		return FetchBlockDataFromHTTP(ctx, item)
	default:
		return nil, xerrors.Errorf("%q not yet supported", u.Scheme)
	}
}

func FetchBlockDataFromHTTP(ctx context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	u, err := url.Parse(item.URL())
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return nil, xerrors.Errorf("%q is not http", u.Scheme)
	}

	client := &http.Client{}
	var r *http.Request
	i, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	r = i.WithContext(ctx)

	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	switch res.StatusCode {
	case http.StatusOK:
		i, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, xerrors.Errorf("failed to request blockdata: %w", err)
		}
		return io.NopCloser(bytes.NewBuffer(i)), nil
	default:
		return nil, xerrors.Errorf("failed to request blockdata: %q", res.Status)
	}
}
