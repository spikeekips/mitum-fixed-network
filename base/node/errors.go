package node

import "github.com/spikeekips/mitum/base"

type NodeError struct {
	err  error
	node base.Address
}

func NewNodeError(no base.Address, err error) NodeError {
	return NodeError{
		node: no,
		err:  err,
	}
}

func (er NodeError) Error() string {
	return er.err.Error()
}

func (er NodeError) Unwrap() error {
	return er.err
}
