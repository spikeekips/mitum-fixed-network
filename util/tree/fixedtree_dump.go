package tree

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/btcsuite/btcutil/base58"
)

func (ft FixedTree) Dump(w io.Writer) error {
	if ft.IsEmpty() {
		return nil
	}

	_, _ = fmt.Fprintf(w, "# %s\n", ft.Hint().String())
	if err := ft.Traverse(func(_ int, key, h, v []byte) (bool, error) {
		_, _ = fmt.Fprintln(w, base58.Encode(key))
		_, _ = fmt.Fprintln(w, base58.Encode(h))
		_, _ = fmt.Fprintln(w, base58.Encode(v))

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

		nodes = append(nodes, base58.Decode(l))
	}
	if err := scanner.Err(); err != nil {
		return FixedTree{}, err
	}

	return NewFixedTree(nodes, nil)
}
