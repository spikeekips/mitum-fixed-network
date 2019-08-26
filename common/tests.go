// +build test

package common

import (
	"runtime/debug"
)

func DebugPanic() {
	if r := recover(); r != nil {
		debug.PrintStack()
		panic(r)
	}
}
