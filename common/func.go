package common

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

var (
	basePath string
)

func init() {
	basePath = filepath.Dir(reflect.TypeOf(basePath).PkgPath())
}

func FuncName(f interface{}, full bool) string {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		return v.String()
	}

	rf := runtime.FuncForPC(v.Pointer())
	if rf == nil {
		return v.String()
	}

	if full {
		return rf.Name()
	}

	if !strings.HasPrefix(rf.Name(), basePath) {
		return rf.Name()
	}
	return rf.Name()[len(basePath)+1:]
}
