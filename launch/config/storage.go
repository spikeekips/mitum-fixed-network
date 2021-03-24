package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spikeekips/mitum/util/cache"
)

var (
	DefaultBlockDataPath    = "./blockdata"
	DefaultMainStorageURI   = "mongodb://127.0.0.1:27017/mitum"
	DefaultMainStorageCache = fmt.Sprintf(
		"gcache:?type=%s&size=%d&expire=%s",
		cache.DefaultGCacheType,
		cache.DefaultGCacheSize,
		cache.DefaultCacheExpire.String(),
	)
)

type BlockData interface {
	Path() string
	SetPath(string) error
}

type BaseBlockData struct {
	path string
}

func (no BaseBlockData) Path() string {
	return no.path
}

func (no *BaseBlockData) SetPath(s string) error {
	no.path = strings.TrimSpace(s)

	return nil
}

type MainStorage interface {
	URI() *url.URL
	SetURI(string) error
	Cache() *url.URL
	SetCache(string) error
}

type BaseMainStorage struct {
	uri   *url.URL
	cache *url.URL
}

func (no BaseMainStorage) URI() *url.URL {
	return no.uri
}

func (no *BaseMainStorage) SetURI(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else {
		no.uri = u

		return nil
	}
}

func (no BaseMainStorage) Cache() *url.URL {
	return no.cache
}

func (no *BaseMainStorage) SetCache(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else {
		no.cache = u

		return nil
	}
}

type Storage interface {
	Main() MainStorage
	SetMain(MainStorage) error
	BlockData() BlockData
	SetBlockData(BlockData) error
}

type BaseStorage struct {
	main      MainStorage
	blockData BlockData
}

func EmptyBaseStorage() *BaseStorage {
	return &BaseStorage{
		main:      &BaseMainStorage{},
		blockData: &BaseBlockData{},
	}
}

func (no BaseStorage) Main() MainStorage {
	return no.main
}

func (no *BaseStorage) SetMain(main MainStorage) error {
	no.main = main

	return nil
}

func (no BaseStorage) BlockData() BlockData {
	return no.blockData
}

func (no *BaseStorage) SetBlockData(bs BlockData) error {
	no.blockData = bs

	return nil
}
