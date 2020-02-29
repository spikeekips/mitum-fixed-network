package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
}

func printError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %v", err)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	printUsage()
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printError(fmt.Errorf("<directory> must be given"))
	}

	directory := os.Args[1]
	if fi, err := os.Stat(directory); os.IsNotExist(err) {
		printError(err)
	} else if !fi.IsDir() {
		printError(fmt.Errorf("%s is not directory", directory))
	}

	directories := map[string]struct{}{}
	err := filepath.Walk(directory, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		if !strings.HasSuffix(fi.Name(), ".go") {
			return nil
		}

		d := filepath.Dir(path)
		if _, found := directories[d]; found {
			return nil
		}

		directories[d] = struct{}{}

		return nil
	})
	if err != nil {
		printError(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(directories))

	ch := make(chan string)
	for d := range directories {
		go func(d string) {
			fset := token.NewFileSet()
			if err := parse(ch, fset, d); err != nil {
				printError(err)
			}
			wg.Done()
		}(d)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var i int
	for c := range ch {
		_, _ = fmt.Fprintln(os.Stdout, c)
		i++
	}

	os.Exit(1)
}

func parse(ch chan string, fset *token.FileSet, directory string) error {
	packages, err := parser.ParseDir(
		fset,
		directory,
		func(os.FileInfo) bool {
			return true
		},
		parser.ParseComments,
	)
	if err != nil {
		return err
	}

	for _, p := range packages {
		for _, f := range p.Files {
			for _, cg := range f.Comments {
				for _, c := range cg.List {
					s := c.Text
					if strings.HasPrefix(s, "//") {
						s = strings.Replace(s, "//", "", 1)
					} else if strings.HasPrefix(s, "/*") {
						s = strings.Replace(
							strings.Replace(s, "/*", "", 1),
							"*/",
							"",
							1,
						)
					}

					ch <- strings.TrimSpace(s)
				}
			}
		}
	}

	return nil
}
