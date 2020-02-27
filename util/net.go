package util

import (
	"net"
	"time"

	"golang.org/x/xerrors"
)

func FreePort(proto string) (int, error) {
	switch proto {
	case "udp":
		addr, err := net.ResolveUDPAddr(proto, "127.0.0.1:0")
		if err != nil {
			return 0, err
		}

		l, err := net.ListenUDP(proto, addr)
		if err != nil {
			return 0, err
		}

		defer func() {
			_ = l.Close()
		}()

		return l.LocalAddr().(*net.UDPAddr).Port, nil
	case "tcp":
		addr, err := net.ResolveTCPAddr(proto, "127.0.0.1:0")
		if err != nil {
			return 0, err
		}

		l, err := net.ListenTCP(proto, addr)
		if err != nil {
			return 0, err
		}

		defer func() {
			_ = l.Close()
		}()

		return l.Addr().(*net.TCPAddr).Port, nil
	default:
		return 0, xerrors.Errorf("invalid proto: udp, tcp")
	}
}

func CheckPort(proto string, addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout(proto, addr, timeout)
	if err != nil {
		return err
	} else if conn == nil {
		return xerrors.Errorf("connection is empty")
	}

	defer func() {
		_ = conn.Close()
	}()

	return nil
}
