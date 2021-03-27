package util

import "github.com/spikeekips/mitum/util/errors"

// NOTE Generaal Errors

var IgnoreError = errors.NewError("ignore")

// Data Errors

var (
	NotFoundError   = errors.NewError("not found")
	FoundError      = errors.NewError("found")
	DuplicatedError = errors.NewError("duplicated error")
)
