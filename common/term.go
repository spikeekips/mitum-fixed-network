package common

import (
	"syscall"
	"unsafe"
)

type winsize struct {
	_   uint16
	Col uint16
	_   uint16
	_   uint16
}

func TermWidth() int {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	if int(retCode) == -1 {
		panic(errno)
	}

	return int(ws.Col)
}
