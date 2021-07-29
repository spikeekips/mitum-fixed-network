package memberlist

import "github.com/spikeekips/mitum/util/errors"

var (
	JoinDeclinedError    = errors.NewError("joining declined")
	JoiningCanceledError = errors.NewError("joining canceled")
)
