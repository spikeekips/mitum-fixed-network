package isaac

import "golang.org/x/xerrors"

// NetworkID will be used to separate mitum network with the other network. For
// exampke, with different NetworkID, same Seal messsage will have different
// hash.
type NetworkID []byte

const maxNetworkIDLength = 300

func (ni NetworkID) IsValid([]byte) error {
	if len(ni) > maxNetworkIDLength {
		return xerrors.Errorf(
			"length of NetworkID too long; max=%d, but len=%d",
			maxNetworkIDLength,
			len(ni),
		)
	}

	return nil
}
