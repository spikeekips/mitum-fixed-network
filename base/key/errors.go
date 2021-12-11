package key

import (
	"github.com/spikeekips/mitum/util"
)

var (
	InvalidKeyError                  = util.NewError("invalid key")
	SignatureVerificationFailedError = util.NewError("signature verification failed")
)
