package isaac

import "github.com/inconshreveable/log15"

var log log15.Logger = log15.New("module", "isaac")

func Log() log15.Logger {
	return log
}
