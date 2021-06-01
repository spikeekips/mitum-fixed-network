package deploy

import (
	"time"
)

func (dk *DeployKey) unpack(uk string, uaa time.Time) error {
	dk.k = uk
	dk.addedAt = uaa

	return nil
}
