package common

import "github.com/spikeekips/mitum/errors"

var LongRunningCommandError = errors.NewError("this command needs to be blocked")
