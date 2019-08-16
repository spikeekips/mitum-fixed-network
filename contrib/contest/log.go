package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"golang.org/x/sync/syncmap"
)

var log log15.Logger = log15.New("module", "contest-main")

var logs *syncmap.Map

func init() {
	logs = &syncmap.Map{}
}

func openFile(directory, n string) io.Writer {
	f := filepath.Join(directory, fmt.Sprintf("%s.log", n))
	l, found := logs.Load(f)
	if found {
		return l.(io.Writer)
	}

	w, err := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	logs.Store(f, w)
	return w
}

func LogFileByNodeHandler(directory string, fmtr log15.Format, quiet bool) log15.Handler {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err := os.MkdirAll(directory, os.ModePerm); err != nil {
			panic(err)
		}
	}

	findNode := func(c []interface{}) string {
		for i := 0; i < len(c); i += 2 {
			if n, ok := c[i].(string); !ok {
				continue
			} else if n == "node" {
				return c[i+1].(string)
			}
		}

		return ""
	}

	h := log15.FuncHandler(func(r *log15.Record) error {
		var w io.Writer
		n := findNode(r.Ctx)
		if len(n) < 1 {
			w = openFile(directory, "none")
		} else {
			w = openFile(directory, n)
		}

		_, err := w.Write(fmtr.Format(r))
		return err
	})

	if !quiet {
		h = log15.MultiHandler(h, log15.StreamHandler(os.Stdout, fmtr))
	}

	return closingHandler{Handler: log15.LazyHandler(log15.SyncHandler(h)), files: logs}
}

type closingHandler struct {
	log15.Handler
	files *syncmap.Map
}

func (h *closingHandler) Close() error {
	h.files.Range(func(_, value interface{}) bool {
		_ = value.(io.Closer).Close()
		return true
	})

	return nil
}
