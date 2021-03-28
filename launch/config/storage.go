package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spikeekips/mitum/util/cache"
)

var (
	DefaultBlockDataPath = "./blockdata"
	DefaultDatabaseURI   = "mongodb://127.0.0.1:27017/mitum"
	DefaultDatabaseCache = fmt.Sprintf(
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

type Database interface {
	URI() *url.URL
	SetURI(string) error
	Cache() *url.URL
	SetCache(string) error
}

type BaseDatabase struct {
	uri   *url.URL
	cache *url.URL
}

func (no BaseDatabase) URI() *url.URL {
	return no.uri
}

func (no *BaseDatabase) SetURI(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else {
		no.uri = u

		return nil
	}
}

func (no BaseDatabase) Cache() *url.URL {
	return no.cache
}

func (no *BaseDatabase) SetCache(s string) error {
	if u, err := ParseURLString(s, true); err != nil {
		return err
	} else if _, err := cache.NewCacheFromURI(u.String()); err != nil {
		return err
	} else {
		no.cache = u

		return nil
	}
}

type Storage interface {
	Database() Database
	SetDatabase(Database) error
	BlockData() BlockData
	SetBlockData(BlockData) error
}

type BaseStorage struct {
	database  Database
	blockData BlockData
}

func EmptyBaseStorage() *BaseStorage {
	return &BaseStorage{
		database:  &BaseDatabase{},
		blockData: &BaseBlockData{},
	}
}

func (no BaseStorage) Database() Database {
	return no.database
}

func (no *BaseStorage) SetDatabase(database Database) error {
	no.database = database

	return nil
}

func (no BaseStorage) BlockData() BlockData {
	return no.blockData
}

func (no *BaseStorage) SetBlockData(bs BlockData) error {
	no.blockData = bs

	return nil
}
