package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spikeekips/mitum/util/cache"
)

var (
	DefaultBlockFSPath      = "./blockfs"
	DefaultBlockFSWideOpen  = false
	DefaultMainStorageURI   = "mongodb://127.0.0.1:27017/mitum"
	DefaultMainStorageCache = fmt.Sprintf(
		"gcache:?type=%s&size=%d&expire=%s",
		cache.DefaultGCacheType,
		cache.DefaultGCacheSize,
		cache.DefaultCacheExpire.String(),
	)
)

type BlockFS interface {
	Path() string
	SetPath(string) error
	WideOpen() bool
	SetWideOpen(bool) error
}

type BaseBlockFS struct {
	path     string
	wideOpen bool
}

func (no BaseBlockFS) Path() string {
	return no.path
}

func (no *BaseBlockFS) SetPath(s string) error {
	no.path = strings.TrimSpace(s)

	return nil
}

func (no BaseBlockFS) WideOpen() bool {
	return no.wideOpen
}

func (no *BaseBlockFS) SetWideOpen(s bool) error {
	no.wideOpen = s

	return nil
}

type MainStorage interface {
	// TODO needs another proper name
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
	BlockFS() BlockFS
	SetBlockFS(BlockFS) error
}

type BaseStorage struct {
	main    MainStorage
	blockFS BlockFS
}

func EmptyBaseStorage() *BaseStorage {
	return &BaseStorage{
		main:    &BaseMainStorage{},
		blockFS: &BaseBlockFS{},
	}
}

func (no BaseStorage) Main() MainStorage {
	return no.main
}

func (no *BaseStorage) SetMain(main MainStorage) error {
	no.main = main

	return nil
}

func (no BaseStorage) BlockFS() BlockFS {
	return no.blockFS
}

func (no *BaseStorage) SetBlockFS(bs BlockFS) error {
	no.blockFS = bs

	return nil
}
