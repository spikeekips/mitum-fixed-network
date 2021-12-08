package key

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	InvalidKeyError                  = isvalid.InvalidError.Errorf("invalid key")
	SignatureVerificationFailedError = util.NewError("signature verification failed")
)
