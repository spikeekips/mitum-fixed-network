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
	var u *url.URL
	if i, err := url.Parse(item.URL()); err != nil {
		return nil, err
	} else {
		u = i
	}

	var ro io.Reader
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
		return nil, xerrors.Errorf("%q not yet supported", u.Scheme)
	}

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

func FetchBlockDataFromRemote(ctx context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	var u *url.URL
	if i, err := url.Parse(item.URL()); err != nil {
		return nil, err
	} else {
		u = i
	}

	switch u.Scheme {
	case "file":
		return nil, xerrors.Errorf("%q is not remote", u.String())
	case "http", "https":
		return FetchBlockDataFromHTTP(ctx, item)
	default:
		return nil, xerrors.Errorf("%q not yet supported", u.Scheme)
	}
}

func FetchBlockDataFromHTTP(ctx context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	var u *url.URL
	if i, err := url.Parse(item.URL()); err != nil {
		return nil, err
	} else {
		u = i
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return nil, xerrors.Errorf("%q is not http", u.Scheme)
	}

	client := &http.Client{}
	var r *http.Request
	if i, err := http.NewRequest("GET", u.String(), nil); err != nil {
		return nil, err
	} else {
		r = i.WithContext(ctx)
	}

	var res *http.Response
	if i, err := client.Do(r); err != nil {
		return nil, err
	} else {
		defer func() {
			_ = i.Body.Close()
		}()

		res = i
	}

	switch res.StatusCode {
	case http.StatusOK:
		if i, err := ioutil.ReadAll(res.Body); err != nil {
			return nil, xerrors.Errorf("failed to request blockdata: %w", err)
		} else {
			return io.NopCloser(bytes.NewBuffer(i)), nil
		}
	default:
		return nil, xerrors.Errorf("failed to request blockdata: %q", res.Status)
	}
}
