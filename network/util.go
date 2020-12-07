package network

import (
	"net"
	"time"

	"golang.org/x/xerrors"
)

func CheckBindIsOpen(network, bind string, timeout time.Duration) error {
	errchan := make(chan error)
	switch network {
	case "tcp":
		go func() {
			if server, err := net.Listen(network, bind); err != nil {
				errchan <- err
			} else if server != nil {
				_ = server.Close()
			}
		}()
	case "udp":
		go func() {
			if server, err := net.ListenPacket(network, bind); err != nil {
				errchan <- err
			} else if server != nil {
				_ = server.Close()
			}
		}()
	}

	select {
	case err := <-errchan:
		return xerrors.Errorf("failed to open bind: %w", err)
	case <-time.After(timeout):
		return nil
	}
}
