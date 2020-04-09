package hint

import (
	"fmt"
	"sort"
	"sync"

	"golang.org/x/mod/semver"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	HintAlreadyAddedError = errors.NewError("hint already added in Hintset")
	HintNotFoundError     = errors.NewError("Hint not found in Hintset")
)

// Hintset is the collection of Hinter. It supports to compare semver for
// Hint.Version().
type Hintset struct {
	m       map[Type][]Hinter
	known   *sync.Map
	cache   *sync.Map
	unknown *sync.Map
}

// NewHintset returns new Hintset.
func NewHintset() *Hintset {
	return &Hintset{
		m:       map[Type][]Hinter{},
		known:   &sync.Map{},
		cache:   &sync.Map{},
		unknown: &sync.Map{},
	}
}

func (st Hintset) key(t Type, v Version) string {
	if len(v) < 1 {
		return fmt.Sprintf("%x", t.Bytes())
	}

	return fmt.Sprintf("%x-%s", t.Bytes(), v)
}

// Add adds new Hint. The same hints will be sorted by version.
func (st *Hintset) Add(hd Hinter) error {
	h := hd.Hint()
	if !isRegisteredType(h.Type()) {
		return UnknownTypeError.Errorf("type=%s", h.Type().Verbose())
	}

	key := st.key(h.Type(), h.Version())
	if _, found := st.known.Load(key); found {
		return HintAlreadyAddedError.Errorf("type=%s version=%s", h.Type().Verbose(), h.Version())
	}

	st.known.Store(key, hd)

	sl := st.m[h.Type()]
	sl = append(sl, hd)
	sort.SliceStable(
		sl,
		func(i, j int) bool {
			return semver.Compare(
				sl[i].Hint().Version().GO(),
				sl[j].Hint().Version().GO(),
			) < 0
		},
	)

	st.m[h.Type()] = sl

	return nil
}

// Remove removes Hint by it's Type() and Version().
func (st *Hintset) Remove(t Type, version Version) error {
	key := st.key(t, version)
	if _, found := st.known.Load(key); !found {
		return HintNotFoundError.Errorf("type=%s version=%s", t.Verbose(), version)
	}

	st.known.Delete(key)

	if len(version) > 0 {
		var sl []Hinter
		for _, hd := range st.m[t] {
			if version == hd.Hint().Version() {
				continue
			}
			sl = append(sl, hd)
		}
		st.m[t] = sl
	}

	// Remove from cache
	st.cache.Range(func(k, v interface{}) bool {
		hd := v.(Hinter)
		if hd.Hint().Type() == t && hd.Hint().Version() == version {
			st.cache.Delete(k)
		}

		return true
	})

	return nil
}

// Hinter finds Hint by Hint.Type() and Hint.Version().
// - If there is no matched Hinter by Type() and Version(), Hinter() will try to
// find same Type() and latest version().
// - If version argument is empty, the latest version of same type will be
// returned.
// - The unknown Hinter will be cached.
func (st *Hintset) Hinter(t Type, version Version) (Hinter, error) {
	key := st.key(t, version)
	if e, found := st.known.Load(key); found {
		return e.(Hinter), nil
	} else if e, found := st.cache.Load(key); found {
		return e.(Hinter), nil
	} else if _, found := st.unknown.Load(key); found {
		return nil, HintNotFoundError.Errorf("failed to find; type=%s version=%s", t.Verbose(), version)
	}

	var hd Hinter
	if len(version) < 1 {
		if l, found := st.m[t]; found {
			hd = l[len(st.m[t])-1]
		}
	} else {
		// NOTE trying to find hint, which is,
		// - same major version
		// - and latest version
		major := semver.Major(version.GO())
	end:
		for _, e := range st.m[t] {
			switch c := semver.Compare(major, semver.Major(e.Hint().Version().GO())); {
			case c < 0:
				continue
			case c > 0:
				break end
			}
			hd = e
		}
	}

	if hd == nil {
		st.unknown.Store(key, hd)
		return nil, HintNotFoundError.Errorf("failed to find; type=%s version=%s", t.Verbose(), version)
	}

	st.cache.Store(key, hd)

	return hd, nil
}

func (st *Hintset) Hinters() map[Type][]Hinter {
	return st.m
}
