package memberlist

import "github.com/spikeekips/mitum/util"

var (
	JoinDeclinedError    = util.NewError("joining declined")
	JoiningCanceledError = util.NewError("joining canceled")
)
