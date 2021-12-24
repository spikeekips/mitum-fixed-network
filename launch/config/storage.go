package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
)

var (
	DefaultBlockdataPath = "./blockdata"
	DefaultDatabaseURI   = "mongodb://127.0.0.1:27017/mitum"
	DefaultDatabaseCache = fmt.Sprintf(
		"gcache:?type=%s&size=%d&expire=%s",
		cache.DefaultGCacheType,
		cache.DefaultGCacheSize,
		cache.DefaultCacheExpire.String(),
	)
	DefaultDatabaseCacheURL *url.URL
)

func init() {
	if i, err := network.ParseURL(DefaultDatabaseCache, false); err != nil {
		panic(err)
	} else {
		DefaultDatabaseCacheURL = i
	}
}

type Blockdata interface {
	Path() string
	SetPath(string) error
}

type BaseBlockdata struct {
	path string
}

func (no BaseBlockdata) Path() string {
	return no.path
}

func (no *BaseBlockdata) SetPath(s string) error {
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
	u, err := network.ParseURL(s, true)
	if err != nil {
		return err
	}
	no.uri = u

	return nil
}

func (no BaseDatabase) Cache() *url.URL {
	return no.cache
}

func (no *BaseDatabase) SetCache(s string) error {
	if u, err := network.ParseURL(s, true); err != nil {
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
	Blockdata() Blockdata
	SetBlockdata(Blockdata) error
}

type BaseStorage struct {
	database  Database
	blockdata Blockdata
}

func EmptyBaseStorage() *BaseStorage {
	return &BaseStorage{
		database: &BaseDatabase{
			cache: DefaultDatabaseCacheURL,
		},
		blockdata: &BaseBlockdata{},
	}
}

func (no BaseStorage) Database() Database {
	return no.database
}

func (no *BaseStorage) SetDatabase(database Database) error {
	no.database = database

	return nil
}

func (no BaseStorage) Blockdata() Blockdata {
	return no.blockdata
}

func (no *BaseStorage) SetBlockdata(bs Blockdata) error {
	no.blockdata = bs

	return nil
}
