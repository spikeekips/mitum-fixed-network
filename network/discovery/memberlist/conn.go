package memberlist

import (
	"bytes"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
)

var connPool = sync.Pool{
	New: func() interface{} {
		return new(QuicConn)
	},
}

var (
	newQuicConn = func(
		id,
		addr string,
		writer func([]byte) error,
		closefunc func() error,
	) (*QuicConn, error) {
		raddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return nil, err
		}

		conn := connPool.Get().(*QuicConn)

		conn.Lock()
		defer conn.Unlock()

		conn.id = id
		conn.addr = addr
		conn.raddr = raddr
		conn.writer = writer
		conn.closefunc = closefunc
		conn.b = bytes.NewBuffer(nil)
		conn.isClosed = false
		conn.readch = make(chan bool)
		conn.createdAt = localtime.UTCNow()

		return conn, nil
	}
	connPoolPut = func(conn *QuicConn) {
		conn.id = ""
		conn.addr = ""
		conn.raddr = nil
		conn.writer = nil
		conn.closefunc = nil
		conn.b = nil

		connPool.Put(conn)
	}
)

type QuicConn struct {
	sync.RWMutex
	id        string
	addr      string
	raddr     *net.TCPAddr
	writer    func([]byte) error
	closefunc func() error
	b         *bytes.Buffer
	isClosed  bool
	readch    chan bool
	createdAt time.Time
}

func (conn *QuicConn) Read(b []byte) (int, error) {
	<-conn.readch

	return conn.b.Read(b)
}

func (conn *QuicConn) Write(b []byte) (int, error) {
	return len(b), conn.writer(b)
}

func (conn *QuicConn) Close() error {
	conn.Lock()
	defer conn.Unlock()

	defer connPoolPut(conn)

	conn.isClosed = true

	if conn.closefunc != nil {
		return conn.closefunc()
	}

	return nil
}

func (*QuicConn) LocalAddr() net.Addr {
	return nil
}

func (conn *QuicConn) RemoteAddr() net.Addr {
	return conn.raddr
}

func (*QuicConn) SetDeadline(time.Time) error {
	return nil
}

func (*QuicConn) SetReadDeadline(time.Time) error {
	return nil
}

func (*QuicConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (conn *QuicConn) append(b []byte) (int, error) {
	n, err := conn.b.Write(b)
	go func() {
		conn.readch <- true
	}()

	return n, err
}

type ConnInfo struct {
	network.HTTPConnInfo
	Address       string
	LastActivated time.Time
}

func NewConnInfo(addr string, publish *url.URL, insecure bool) ConnInfo {
	return NewConnInfoWithConnInfo(addr, network.NewHTTPConnInfo(publish, insecure))
}

func NewConnInfoWithConnInfo(addr string, connInfo network.HTTPConnInfo) ConnInfo {
	return ConnInfo{
		HTTPConnInfo:  connInfo,
		Address:       addr,
		LastActivated: localtime.UTCNow(),
	}
}

func (ci ConnInfo) IsValid([]byte) error {
	if err := ci.HTTPConnInfo.IsValid(nil); err != nil {
		return err
	}

	if len(strings.TrimSpace(ci.Address)) < 1 {
		return errors.Errorf("empty address")
	}

	return isValidPublishURL(ci.URL())
}

func (ci ConnInfo) Equal(b network.ConnInfo) bool {
	if b == nil {
		return false
	}

	i, ok := b.(ConnInfo)
	if !ok {
		return false
	}

	if ci.Address != i.Address {
		return false
	}

	return ci.HTTPConnInfo.Equal(i.HTTPConnInfo)
}

func (ci ConnInfo) Bytes() []byte {
	return util.ConcatBytesSlice(
		ci.HTTPConnInfo.Bytes(),
		[]byte(ci.Address),
	)
}

func (ci ConnInfo) setInsecure(i bool) ConnInfo {
	ci.HTTPConnInfo = ci.HTTPConnInfo.SetInsecure(i)

	return ci
}

type ConnInfoMap struct {
	sync.RWMutex
	addrs *sync.Map
}

func NewConnInfoMap() *ConnInfoMap {
	return &ConnInfoMap{
		addrs: &sync.Map{},
	}
}

func (ma *ConnInfoMap) addrExists(addr string) bool {
	_, found := ma.addrs.Load(addr)

	return found
}

func (ma *ConnInfoMap) dryAdd(u *url.URL, insecure bool) (ConnInfo, error) {
	uu, addr, err := publishToAddress(u)
	if err != nil {
		return ConnInfo{}, errors.Wrapf(err, "failed to convert url to address, %q", u.String())
	}

	item := NewConnInfo(addr, uu, insecure)

	ma.store(item)

	return item, nil
}

// Add adds new node and returns ConnInfo, which has unique net.Addr string
// for memberlist.Address.
func (ma *ConnInfoMap) add(u *url.URL, insecure bool) (ConnInfo, error) {
	ma.Lock()
	defer ma.Unlock()

	uu, addr, err := publishToAddress(u)
	if err != nil {
		return ConnInfo{}, errors.Wrapf(err, "failed to convert url to address, %q", u.String())
	}

	if i, found := ma.load(addr); found { // NOTE if already exists, compare item values
		if i.Insecure() != insecure {
			i = i.setInsecure(insecure)
		}

		i.LastActivated = localtime.Now()
		ma.store(i)

		return i, nil
	}

	item := NewConnInfo(addr, uu, insecure)

	ma.store(item)

	return item, nil
}

// Item returns ConnInfo
func (ma *ConnInfoMap) item(addr string) (ConnInfo, bool) {
	ma.RLock()
	defer ma.RUnlock()

	i, found := ma.addrs.Load(addr)
	if !found {
		return ConnInfo{}, false
	}

	return i.(ConnInfo), true
}

func (ma *ConnInfoMap) remove(addr string) bool {
	ma.Lock()
	defer ma.Unlock()

	if _, found := ma.addrs.Load(addr); !found {
		return false
	}

	ma.addrs.Delete(addr)

	return true
}

func (ma *ConnInfoMap) traverse(f func(ConnInfo) bool) {
	ma.addrs.Range(func(k, v interface{}) bool {
		return f(v.(ConnInfo))
	})
}

func (ma *ConnInfoMap) setAlive(addr string) error {
	ma.Lock()
	defer ma.Unlock()

	i, found := ma.addrs.Load(addr)
	if !found {
		return errors.Errorf("address, %q not found", addr)
	}

	j := i.(ConnInfo)
	j.LastActivated = localtime.Now()

	ma.store(j)

	return nil
}

func (ma *ConnInfoMap) load(addr string) (ConnInfo, bool) {
	i, found := ma.addrs.Load(addr)
	if !found {
		return ConnInfo{}, false
	}

	return i.(ConnInfo), true
}

func (ma *ConnInfoMap) store(item ConnInfo) {
	ma.addrs.Store(item.Address, item)
}

type NodeConnInfo struct {
	ConnInfo
	node base.Address
}

func NewNodeConnInfo(connInfo ConnInfo, node base.Address) NodeConnInfo {
	return NodeConnInfo{ConnInfo: connInfo, node: node}
}

func (conn NodeConnInfo) Node() base.Address {
	return conn.node
}
