package network

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NilConnInfoType    = hint.Type("nil-conninfo")
	NilConnInfoHint    = hint.NewHint(NilConnInfoType, "v0.0.1")
	NilConnInfoHinter  = NilConnInfo{BaseHinter: hint.NewBaseHinter(NilConnInfoHint)}
	HTTPConnInfoType   = hint.Type("http-conninfo")
	HTTPConnInfoHint   = hint.NewHint(HTTPConnInfoType, "v0.0.1")
	HTTPConnInfoHinter = HTTPConnInfo{BaseHinter: hint.NewBaseHinter(HTTPConnInfoHint)}
)

type ConnInfo interface {
	hint.Hinter
	isvalid.IsValider
	fmt.Stringer
	util.Byter
	URL() *url.URL
	Insecure() bool
	Equal(ConnInfo) bool
}

type NilConnInfo struct {
	hint.BaseHinter
	s string
}

func NewNilConnInfo(name string) NilConnInfo {
	return NilConnInfo{
		BaseHinter: hint.NewBaseHinter(NilConnInfoHint),
		s:          fmt.Sprintf("<nil ConnInfo>: %s", name),
	}
}

func (conn NilConnInfo) IsValid([]byte) error {
	if err := conn.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if len(strings.TrimSpace(conn.s)) < 1 {
		return util.EmptyError.Errorf("NilConnInfo")
	}

	return nil
}

func (NilConnInfo) URL() *url.URL {
	return nil
}

func (NilConnInfo) Insecure() bool {
	return false
}

func (conn NilConnInfo) Equal(b ConnInfo) bool {
	if b == nil {
		return false
	}

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
	hint.BaseHinter
	u        *url.URL
	insecure bool
}

func NewHTTPConnInfo(u *url.URL, insecure bool) HTTPConnInfo {
	return HTTPConnInfo{
		BaseHinter: hint.NewBaseHinter(HTTPConnInfoHint),
		u:          NormalizeURL(u), insecure: insecure,
	}
}

func NewHTTPConnInfoFromString(s string, insecure bool) (HTTPConnInfo, error) {
	u, err := NormalizeURLString(s)
	if err != nil {
		return HTTPConnInfo{}, errors.Wrapf(err, "wrong node url, %q", s)
	}

	return NewHTTPConnInfo(u, insecure), nil
}

func (conn HTTPConnInfo) IsValid([]byte) error {
	if err := conn.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	return IsValidURL(conn.u)
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
	if b == nil {
		return false
	}

	i, ok := b.(HTTPConnInfo)
	if !ok {
		return false
	}

	if conn.insecure != i.insecure {
		return false
	}

	switch {
	case conn.u == nil && i.u != nil:
		return false
	case conn.u != nil && i.u == nil:
		return false
	case !reflect.DeepEqual(conn.u.Query(), i.u.Query()):
		return false
	case conn.u.Scheme != i.u.Scheme:
		return false
	case conn.u.User != i.u.User:
		return false
	case conn.u.Host != i.u.Host:
		return false
	case conn.u.Path != i.u.Path:
		return false
	case conn.u.Fragment != i.u.Fragment:
		return false
	}

	return true
}

func (conn HTTPConnInfo) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(conn.u.String()),
		util.BoolToBytes(conn.insecure),
	)
}

func (conn HTTPConnInfo) String() string {
	s := conn.u.String()
	if conn.insecure {
		s += "#insecure"
	}

	return s
}
