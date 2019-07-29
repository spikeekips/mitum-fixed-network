package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
)

var log log15.Logger = log15.New("module", "contest-main")

func LogFileByNodeHandler(directory string, fmtr log15.Format, quiet bool) log15.Handler {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err := os.MkdirAll(directory, os.ModePerm); err != nil {
			panic(err)
		}
	}

	logs := map[string]*os.File{}

	openFile := func(n string) io.Writer {
		f := filepath.Join(directory, fmt.Sprintf("%s.log", n))
		w, err := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}

		logs[n] = w
		return w
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

	_ = openFile("none")

	h := log15.FuncHandler(func(r *log15.Record) error {
		var w io.Writer
		n := findNode(r.Ctx)
		if len(n) < 1 {
			w = logs["none"]
		} else if f, found := logs[n]; found {
			w = f
		} else {
			w = openFile(n)
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
	files map[string]*os.File
}

func (h *closingHandler) Close() error {
	for _, f := range h.files {
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
