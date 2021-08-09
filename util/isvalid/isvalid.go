package isvalid

import "github.com/spikeekips/mitum/util"

var InvalidError = util.NewError("invalid")

type IsValider interface {
	IsValid([]byte) error
}
