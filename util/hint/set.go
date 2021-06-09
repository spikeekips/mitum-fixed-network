package hint

import (
	"sort"
	"sync"

	"github.com/bluele/gcache"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"
)

type Hintset struct {
	sync.RWMutex
	set   map[Type][]Hinter
	m     map[string]Hinter
	cache gcache.Cache
}

func NewHintset() *Hintset {
	return &Hintset{
		set:   map[Type][]Hinter{},
		m:     map[string]Hinter{},
		cache: gcache.New(100 * 100).LRU().Build(),
	}
}

func (hs *Hintset) Add(ht Hinter) error {
	if err := ht.Hint().IsValid(nil); err != nil {
		return err
	}

	if _, found := hs.m[ht.Hint().String()]; found {
		return util.FoundError.Errorf("Hint already added: %q", ht)
	} else {
		hs.m[ht.Hint().String()] = ht
	}

	l := hs.set[ht.Hint().Type()]
	l = append(l, ht)

	sort.SliceStable(l, func(i, j int) bool {
		return semver.Compare(l[i].Hint().Version(), l[j].Hint().Version()) > 0
	})

	hs.set[ht.Hint().Type()] = l

	return nil
}

func (hs *Hintset) Latest(ty Type) (Hinter, error) {
	if i, found := hs.set[ty]; !found {
		return nil, util.NotFoundError.Errorf("Type, %q not found", ty)
	} else {
		return i[0], nil
	}
}

func (hs *Hintset) Get(ht Hint) Hinter {
	return hs.m[ht.String()]
}

func (hs *Hintset) Hinters(ty Type) []Hinter {
	return hs.set[ty]
}

func (hs *Hintset) Compatible(ht Hint) (Hinter, error) {
	if i, err := hs.cache.Get(ht.String()); err == nil {
		if h, ok := i.(Hinter); ok {
			return h, nil
		} else {
			return nil, util.NotFoundError.Errorf("Hinter not found for %q", ht)
		}
	} else if !xerrors.Is(err, gcache.KeyNotFoundError) {
		return nil, xerrors.Errorf("Hintset cache problem: %w", err)
	}

	hinter := hs.compatible(ht)
	if hinter == nil {
		err := util.NotFoundError.Errorf("Hinter not found for %q", ht)
		_ = hs.cache.Set(ht.String(), err)

		return nil, err
	}

	_ = hs.cache.Set(ht.String(), hinter)

	return hinter, nil
}

func (hs *Hintset) compatible(ht Hint) Hinter {
	l, found := hs.set[ht.Type()]
	if !found {
		return nil
	}

	for i := range l {
		j := l[i]
		if err := j.Hint().IsCompatible(ht); err == nil {
			return j
		}
	}

	return nil
}

type GlobalHintset struct {
	*Hintset
}

func NewGlobalHintset() *GlobalHintset {
	return &GlobalHintset{
		Hintset: NewHintset(),
	}
}

func (hs *GlobalHintset) Initialize() error {
	for i := range hs.set {
		if len(hs.set[i]) < 1 {
			return xerrors.Errorf("empty Type, %q found", i)
		}
	}

	return nil
}

func (hs *GlobalHintset) HasType(ty Type) bool {
	_, found := hs.set[ty]

	return found
}

func (hs *GlobalHintset) AddType(ty Type) error {
	if _, found := hs.set[ty]; found {
		return util.FoundError.Errorf("already added type, %q", ty)
	}

	hs.set[ty] = nil

	return nil
}

func (hs *GlobalHintset) Add(ht Hinter) error {
	if _, found := hs.set[ht.Hint().Type()]; !found {
		return util.NotFoundError.Errorf("unknown type, %q", ht.Hint().Type())
	}

	return hs.Hintset.Add(ht)
}
