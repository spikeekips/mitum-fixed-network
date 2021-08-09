package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
)

type ErrorPointYAML struct {
	Type  string
	Point string
}

func (ep *ErrorPoint) UnmarshalYAML(decode func(v interface{}) error) error {
	var epy ErrorPointYAML
	if err := decode(&epy); err != nil {
		return err
	}

	if et := ErrorType(strings.TrimSpace(epy.Type)); len(et) < 1 {
		ep.Type = ErrorTypeError
	} else if err := et.IsValid(nil); err != nil {
		return err
	} else {
		ep.Type = et
	}

	h, r, err := parseErrorPointPoint(epy.Point)
	if err != nil {
		return err
	}
	ep.Height = h
	ep.Round = r

	return nil
}

func parseErrorPointPoint(s string) (base.Height, base.Round, error) {
	if s = strings.TrimSpace(s); len(s) < 1 {
		return base.NilHeight, 0, errors.Errorf("empty point string")
	}

	var h int64
	var r uint64
	if n, err := fmt.Sscanf(s, "%d,%d", &h, &r); err != nil {
		return base.NilHeight, 0, errors.Wrapf(err, "invalid block point string: %v", s)
	} else if n != 2 {
		return base.NilHeight, 0, errors.Errorf("invalid block point string: %v", s)
	}

	height := base.Height(h)
	if err := height.IsValid(nil); err != nil {
		return base.NilHeight, 0, err
	}

	return height, base.Round(r), nil
}
