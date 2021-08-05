package memberlist

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	ml "github.com/hashicorp/memberlist"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type QuicRequest func(
	ctx context.Context,
	insecure bool,
	timeout time.Duration,
	u, /* url */
	method string,
	body []byte,
	header http.Header,
) (*http.Response, func() error, error)

type QuicTransport struct {
	*logging.Logging
	request         QuicRequest
	newNodeMessage  func([]byte, string) ([]byte, error)
	loadNodeMessage func([]byte) (NodeMessage, error)
	ma              *ConnInfoMap
	packetch        chan *ml.Packet
	streamch        chan net.Conn
	timeoutLock     sync.Mutex
	timeout         time.Duration
	conns           *sync.Map
}

func NewQuicTransport(
	request QuicRequest,
	newNodeMessage func([]byte, string) ([]byte, error),
	loadNodeMessage func([]byte) (NodeMessage, error),
	ma *ConnInfoMap,
	timeout time.Duration,
) *QuicTransport {
	return &QuicTransport{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "memberlist-discovery-transport")
		}),
		request:         request,
		newNodeMessage:  newNodeMessage,
		loadNodeMessage: loadNodeMessage,
		ma:              ma,
		packetch:        make(chan *ml.Packet),
		streamch:        make(chan net.Conn),
		timeout:         timeout,
		conns:           &sync.Map{},
	}
}

func (tp *QuicTransport) getTimeout() time.Duration {
	tp.timeoutLock.Lock()
	defer tp.timeoutLock.Unlock()

	if tp.timeout < 1 {
		tp.timeout = defaultTCPTimeout
	}

	return tp.timeout
}

func (*QuicTransport) FinalAdvertiseAddr(ip string, port int) (net.IP, int, error) {
	return net.ParseIP(ip), port, nil
}

func (tp *QuicTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	a := ml.Address{Addr: addr, Name: ""}

	return tp.WriteToAddress(b, a)
}

func (tp *QuicTransport) WriteToAddress(b []byte, a ml.Address) (time.Time, error) {
	return localtime.UTCNow(), tp.write(a.Addr, b, tp.getTimeout(), "")
}

func (tp *QuicTransport) PacketCh() <-chan *ml.Packet {
	return tp.packetch
}

func (tp *QuicTransport) DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	a := ml.Address{Addr: addr, Name: ""}
	return tp.DialAddressTimeout(a, timeout)
}

func (tp *QuicTransport) DialAddressTimeout(a ml.Address, timeout time.Duration) (net.Conn, error) {
	conn, err := tp.dialAddressTimeout(a, timeout)
	if err != nil {
		return nil, &net.OpError{Net: "tcp", Op: "dial", Err: xerrors.Errorf("dial: failed to connect, %q: %w", a, err)}
	}

	return conn, nil
}

func (tp *QuicTransport) StreamCh() <-chan net.Conn {
	return tp.streamch
}

func (*QuicTransport) Shutdown() error {
	return nil
}

func (tp *QuicTransport) dialAddressTimeout(a ml.Address, timeout time.Duration) (net.Conn, error) {
	ci, found := tp.ma.item(a.Addr)
	if !found {
		tp.Log().Warn().Str("address", a.Addr).Msg("unknown address found")

		return nil, xerrors.Errorf("unknown address, %q", a.Addr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), tp.getTimeout())
	defer cancel()

	res, closefunc, err := tp.request(ctx, ci.Insecure(), timeout, ci.URL().String(), "HEAD", nil, nil)
	switch {
	case err != nil:
		return nil, err
	case res.StatusCode == http.StatusOK:
	case res.StatusCode == http.StatusCreated:
	default:
		if res != nil {
			err = xerrors.Errorf("status=%d", res.StatusCode)
		}

		return nil, err
	}

	defer func() {
		_ = closefunc()
		_ = res.Body.Close()
	}()

	conn, err := tp.newQuicConn(util.UUID().String(), a.Addr, timeout)
	if err != nil {
		return nil, err
	}

	tp.conns.Store(conn.id, conn)

	return conn, nil
}

func (tp *QuicTransport) write(addr string, body []byte, timeout time.Duration, conid string) error {
	ci, found := tp.ma.item(addr)
	if !found {
		tp.Log().Warn().Str("address", addr).Msg("unknown address found")

		return xerrors.Errorf("write: unknown address, %q", addr)
	}

	b, err := tp.newNodeMessage(body, conid)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res, closefunc, err := tp.request(
		ctx,
		ci.Insecure(),
		timeout,
		ci.URL().String(),
		"POST",
		b,
		nil,
	)
	if err != nil {
		return &net.OpError{Net: "tcp", Op: "write", Err: err}
	}
	defer func() {
		_ = closefunc()
		_ = res.Body.Close()
	}()

	switch res.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	default:
		return &net.OpError{
			Net: "tcp", Op: "write",
			Err: xerrors.Errorf("failed to write; status=%d", res.StatusCode),
		}
	}
}

func (tp *QuicTransport) newQuicConn(id, addr string, timeout time.Duration) (*QuicConn, error) {
	return newQuicConn(
		id,
		addr,
		func(b []byte) error {
			return tp.write(addr, b, timeout, id)
		},
		func() error {
			tp.conns.Delete(id)

			return nil
		},
	)
}

func (tp *QuicTransport) handler(callback func(NodeMessage) error) http.HandlerFunc {
	if callback == nil {
		callback = func(NodeMessage) error { return nil }
	}

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.WriteHeader(http.StatusCreated)

			return
		case "POST":
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

			return
		}

		var ms NodeMessage
		if i, err := ioutil.ReadAll(r.Body); err != nil {
			tp.Log().Trace().Err(err).Msg("failed to read body")

			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

			return
		} else if ms, err = tp.loadNodeMessage(i); err != nil {
			tp.Log().Trace().Err(err).Msg("failed to load NodeMessage")

			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

			return
		}

		l := tp.Log().With().
			Str("conn_id", ms.connid).
			Stringer("node", ms.node).
			Interface("conninfo", ms.ConnInfo).
			Logger()

		if err := callback(ms); err != nil {
			l.Trace().Err(err).Msg("failed to handle; ignored")

			switch {
			case xerrors.Is(err, JoinDeclinedError):
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			default:
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			}

			return
		}

		if !tp.ma.addrExists(ms.Address) {
			l.Trace().Msg("address not found in ConnMap; will be added")

			_, _ = tp.ma.add(ms.URL(), ms.Insecure())
		} else if err := tp.ma.setAlive(ms.Address); err != nil {
			l.Trace().Err(err).Msg("failed to set alive")
		}

		if len(ms.connid) > 0 {
			if err := tp.handleStream(ms, l); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

				return
			}
		} else {
			if err := tp.handlePacket(ms, l); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

				return
			}
		}
	}
}

func (tp *QuicTransport) handleStream(ms NodeMessage, l zerolog.Logger) error {
	if i, found := tp.conns.Load(ms.connid); found {
		_, _ = i.(*QuicConn).append(ms.body)

		return nil
	}

	conn, err := tp.newQuicConn(ms.connid, ms.Address, tp.getTimeout())
	if err != nil {
		return err
	}

	_, _ = conn.append(ms.body)

	l.Trace().Str("method", "stream").Int("body", len(ms.body)).Msg("got stream")
	go func() {
		tp.streamch <- conn
	}()

	return nil
}

func (tp *QuicTransport) handlePacket(ms NodeMessage, l zerolog.Logger) error {
	raddr, err := net.ResolveUDPAddr("udp", ms.Address)
	if err != nil {
		return err
	}

	l.Trace().
		Stringer("remote_address", raddr).
		Str("method", "packet").
		Int("body", len(ms.body)).
		Msg("got packet")
	go func() {
		tp.packetch <- &ml.Packet{
			Buf:       ms.body,
			From:      raddr,
			Timestamp: localtime.UTCNow(),
		}
	}()

	return nil
}

func (tp *QuicTransport) checkConnections() {
	now := localtime.UTCNow()

	tp.conns.Range(func(k, v interface{}) bool {
		i := v.(*QuicConn)
		i.RLock()
		defer i.RUnlock()

		tp.Log().Trace().
			Str("conn_id", k.(string)).
			Stringer("created_at", i.createdAt).
			Dur("time_diff", now.Sub(i.createdAt)).
			Msg("connection found")

		return true
	})
}
