package process

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type EmptySuffrage struct{}

func (EmptySuffrage) Initialize() error {
	return nil
}

func (EmptySuffrage) NumberOfActing() uint {
	return 0
}

func (EmptySuffrage) Acting(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	return base.NewActingSuffrage(height, round, nil, nil), nil
}

func (EmptySuffrage) IsInside(base.Address) bool {
	return false
}

func (EmptySuffrage) IsActing(base.Height, base.Round, base.Address) (bool, error) {
	return false, nil
}

func (EmptySuffrage) IsProposer(base.Height, base.Round, base.Address) (bool, error) {
	return false, nil
}

func (EmptySuffrage) Nodes() []base.Address {
	return nil
}

func (EmptySuffrage) Name() string {
	return "empty-suffrage"
}

func (sf EmptySuffrage) Verbose() string {
	m := map[string]interface{}{
		"type": sf.Name(),
	}

	b, err := jsonenc.Marshal(m)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"%+v\n",
			errors.Wrap(err, "failed to marshal EmptySuffrage.Verbose()").Error(),
		)

		return sf.Name()
	}
	return string(b)
}
