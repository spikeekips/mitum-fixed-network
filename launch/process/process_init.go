package process

import (
	"context"

	"github.com/spikeekips/mitum/util"
)

const HookNameCheckVersion = "check_version"

func HookCheckVersion(ctx context.Context) (context.Context, error) {
	var version util.Version
	if err := LoadVersionContextValue(ctx, &version); err != nil {
		return ctx, err
	}

	return ctx, version.IsValid(nil)
}
