package network

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
)

type ConnInfo interface {
	fmt.Stringer
	util.Byter
	URL() *url.URL
	Insecure() bool
	Equal(ConnInfo) bool
}

type NilConnInfo struct {
	s string
}

func NewNilConnInfo(name string) NilConnInfo {
	return NilConnInfo{s: fmt.Sprintf("<nil ConnInfo>: %s", name)}
}

func (NilConnInfo) URL() *url.URL {
	return nil
}

func (NilConnInfo) Insecure() bool {
	return false
}

func (conn NilConnInfo) Equal(b ConnInfo) bool {
	i, ok := b.(NilConnInfo)
	if !ok {
		return false
	}

	return conn.s == i.s
}

func (conn NilConnInfo) String() string {
	return conn.s
}

func (NilConnInfo) Bytes() []byte {
	return nil
}

type HTTPConnInfo struct {
	u        *url.URL
	insecure bool
}

func NewHTTPConnInfo(u *url.URL, insecure bool) HTTPConnInfo {
	return HTTPConnInfo{u: u, insecure: insecure}
}

func NewHTTPConnInfoFromString(s string, insecure bool) (HTTPConnInfo, error) {
	u, err := NormalizeURLString(s)
	if err != nil {
		return HTTPConnInfo{}, errors.Wrapf(err, "wrong node url, %q", s)
	}

	return NewHTTPConnInfo(u, insecure), nil
}

func (conn HTTPConnInfo) URL() *url.URL {
	return conn.u
}

func (conn HTTPConnInfo) Insecure() bool {
	return conn.insecure
}

func (conn HTTPConnInfo) SetInsecure(i bool) HTTPConnInfo {
	conn.insecure = i

	return conn
}

func (conn HTTPConnInfo) Equal(b ConnInfo) bool {
	i, ok := b.(HTTPConnInfo)
	if !ok {
		return false
	}

	switch {
	case conn.u == nil && i.u != nil:
		return false
	case conn.u != nil && i.u == nil:
		return false
	case conn.u.String() != i.u.String():
		return false
	}

	return conn.insecure == i.insecure
}

func (conn HTTPConnInfo) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(conn.u.String()),
		util.BoolToBytes(conn.insecure),
	)
}

func (conn HTTPConnInfo) String() string {
	return conn.u.String()
}

func (conn HTTPConnInfo) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(map[string]interface{}{
		"url":      conn.u.String(),
		"insecure": conn.insecure,
	})
}
