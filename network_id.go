package mitum

import "github.com/spikeekips/mitum/errors"

// NetworkID will be used to separate mitum network with the other network. For
// exampke, with different NetworkID, same Seal messsage will have different
// hash.
type NetworkID []byte

const maxNetworkIDLength = 300

var (
	NetworkIDLengthTooLongError = errors.NewError("length of NetworkID too long: max=%d", maxNetworkIDLength)
)

func (ni NetworkID) IsValid([]byte) error {
	if len(ni) > maxNetworkIDLength {
		return NetworkIDLengthTooLongError.Wrapf("len=%d", len(ni))
	}

	return nil
}
