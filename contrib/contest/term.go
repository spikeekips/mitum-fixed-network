package main

import (
	"syscall"
	"unsafe"
)

func TermWidth() uint {
	ws := &struct {
		_   uint16
		Col uint16
		_   uint16
		_   uint16
	}{}
	retCode, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	if int(retCode) == -1 {
		panic(errno)
	}
	return uint(ws.Col)
}
