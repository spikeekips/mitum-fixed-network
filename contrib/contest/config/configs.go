package contest_config

import (
	"reflect"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type IsValider interface {
	IsValid() error
}

type Merger interface {
	Merge(interface{}) error
}

type nameBasedConfig struct {
	Name string `yaml:"name"`
}

type NameBasedConfig struct {
	nameBasedConfig
	types    map[string]interface{}
	instance interface{}
}

func NewNameBasedConfig(types map[string]interface{}) NameBasedConfig {
	return NameBasedConfig{types: types}
}

func (nc *NameBasedConfig) UnmarshalYAML(value *yaml.Node) error {
	var nnc nameBasedConfig
	if err := value.Decode(&nnc); err != nil {
		return err
	}

	ci, found := nc.types[nnc.Name]
	if !found {
		return xerrors.Errorf("given module not found: %q", nc.Name)
	}

	cii := reflect.New(reflect.TypeOf(ci)).Interface()
	if err := value.Decode(cii); err != nil {
		return err
	} else if cv, ok := cii.(IsValider); !ok {
		return xerrors.Errorf("module does not support IsValider: %q", nnc.Name)
	} else if err := cv.IsValid(); err != nil {
		return err
	} else {
		nc.instance = reflect.ValueOf(cii).Interface()
	}

	nc.Name = nc.Name

	return nil
}

func (nc *NameBasedConfig) Instance() interface{} {
	return nc.instance
}
