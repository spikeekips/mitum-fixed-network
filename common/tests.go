// +build test

package common

import (
	"runtime/debug"

	"github.com/inconshreveable/log15"
)

func DebugPanic() {
	if r := recover(); r != nil {
		debug.PrintStack()
		panic(r)
	}
}

func SetTestLogger(logger log15.Logger) {
	//handler, _ := LogHandler(LogFormatter("terminal"), "")
	handler, _ := LogHandler(LogFormatter("json"), "")
	handler = log15.CallerFileHandler(handler)
	logger.SetHandler(log15.LvlFilterHandler(log15.LvlDebug, handler))
	//logger.SetHandler(log15.LvlFilterHandler(log15.LvlCrit, handler))
}

func init() {
	SetTestLogger(Log())
}
