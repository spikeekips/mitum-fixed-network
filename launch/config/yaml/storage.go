package yamlconfig

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
)

type BlockFS struct {
	Path     *string
	WideOpen *bool `yaml:"wide-open,omitempty"`
}

type MainStorage struct {
	URI   *string `yaml:",omitempty"`
	Cache *string `yaml:",omitempty"`
}

func (no MainStorage) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var conf config.MainStorage
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.Storage().Main()
	}

	if no.URI != nil {
		if err := conf.SetURI(*no.URI); err != nil {
			return ctx, err
		}
	}
	if no.Cache != nil {
		if err := conf.SetCache(*no.Cache); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

type Storage struct {
	Main    *MainStorage `yaml:",inline"`
	BlockFS *BlockFS     `yaml:",omitempty"`
}

func (no Storage) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var conf config.Storage
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.Storage()
	}

	if no.Main != nil {
		if i, err := no.Main.Set(ctx); err != nil {
			return ctx, err
		} else {
			ctx = i
		}
	}

	if no.BlockFS != nil {
		if no.BlockFS.Path != nil {
			if err := conf.BlockFS().SetPath(*no.BlockFS.Path); err != nil {
				return ctx, err
			}
		}
		if no.BlockFS.WideOpen != nil {
			if err := conf.BlockFS().SetWideOpen(*no.BlockFS.WideOpen); err != nil {
				return ctx, err
			}
		}
	}

	return ctx, nil
}
