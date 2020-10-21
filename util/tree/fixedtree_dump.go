package tree

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

func (ft FixedTree) Dump(w io.Writer) error {
	if ft.IsEmpty() {
		return nil
	}

	_, _ = fmt.Fprintf(w, "# %s\n", ft.Hint().String())
	if err := ft.Traverse(func(_ int, key, h, v []byte) (bool, error) {
		_, _ = fmt.Fprintln(w, hex.EncodeToString(key))
		_, _ = fmt.Fprintln(w, hex.EncodeToString(h))
		_, _ = fmt.Fprintln(w, hex.EncodeToString(v))

		return true, nil
	}); err != nil {
		return err
	}

	return nil
}

func LoadFixedTreeFromReader(r io.Reader) (FixedTree, error) {
	// FUTURE support Hint
	var nodes [][]byte

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "# ") {
			continue
		}

		if b, err := hex.DecodeString(l); err != nil {
			return FixedTree{}, err
		} else {
			nodes = append(nodes, b)
		}
	}
	if err := scanner.Err(); err != nil {
		return FixedTree{}, err
	}

	return NewFixedTree(nodes, nil)
}
