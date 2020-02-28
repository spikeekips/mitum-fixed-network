package isaac

import (
	"github.com/spikeekips/mitum/errors"
)

var InvalidProcessStateError = errors.NewError("invalid ProcessState")

type ProcessState uint8

const (
	// ProcessStateBooting means process is just started and ready for proper
	// deployment.
	ProcessStateBooting ProcessState = iota
	// ProcessStateDeployed means process is deployed successfully.
	ProcessStateDeployed
	// ProcessStateStopping means process is stopping.
	ProcessStateStopping
)

func (st ProcessState) String() string {
	switch st {
	case ProcessStateBooting:
		return "BOOTING"
	case ProcessStateDeployed:
		return "DEPLOYED"
	case ProcessStateStopping:
		return "STOPPING"
	default:
		return "<unknown ProcessState>"
	}
}

func (st ProcessState) IsValid([]byte) error {
	switch st {
	case ProcessStateBooting, ProcessStateDeployed, ProcessStateStopping:
		return nil
	}

	return InvalidProcessStateError.Wrapf("ProcessState=%d", st)
}

func (st ProcessState) MarshalText() ([]byte, error) {
	return []byte(st.String()), nil
}
