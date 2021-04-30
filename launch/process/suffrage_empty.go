package process

import (
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type EmptySuffrage struct{}

func (sf EmptySuffrage) Initialize() error {
	return nil
}

func (sf EmptySuffrage) NumberOfActing() uint {
	return 0
}

func (sf EmptySuffrage) Acting(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	return base.NewActingSuffrage(height, round, nil, nil), nil
}

func (sf EmptySuffrage) IsInside(base.Address) bool {
	return false
}

func (sf EmptySuffrage) IsActing(base.Height, base.Round, base.Address) (bool, error) {
	return false, nil
}

func (sf EmptySuffrage) IsProposer(base.Height, base.Round, base.Address) (bool, error) {
	return false, nil
}

func (sf EmptySuffrage) Nodes() []base.Address {
	return nil
}

func (sf EmptySuffrage) Name() string {
	return "empty-suffrage"
}

func (sf EmptySuffrage) Verbose() string {
	m := map[string]interface{}{
		"type": sf.Name(),
	}

	if b, err := jsonenc.Marshal(m); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			xerrors.Errorf("failed to marshal EmptySuffrage.Verbose(): %w", err).Error(),
		)

		return sf.Name()
	} else {
		return string(b)
	}
}
