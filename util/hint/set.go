package hint

import (
	"sort"
	"sync"

	"github.com/bluele/gcache"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/mod/semver"
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

	if _, found := hs.m[ht.Hint().RawString()]; found {
		return util.FoundError.Errorf("Hint already added: %q", ht)
	}
	hs.m[ht.Hint().RawString()] = ht

	l := hs.set[ht.Hint().Type()]
	l = append(l, ht)

	sort.SliceStable(l, func(i, j int) bool {
		return semver.Compare(l[i].Hint().Version(), l[j].Hint().Version()) > 0
	})

	hs.set[ht.Hint().Type()] = l

	return nil
}

func (hs *Hintset) Latest(ty Type) (Hinter, error) {
	i, found := hs.set[ty]
	if !found {
		return nil, util.NotFoundError.Errorf("Type, %q not found", ty)
	}
	return i[0], nil
}

func (hs *Hintset) Get(ht Hint) Hinter {
	return hs.m[ht.RawString()]
}

func (hs *Hintset) Types() []Type {
	l := make([]Type, len(hs.set))

	var i int
	for j := range hs.set {
		l[i] = j
		i++
	}

	return l
}

func (hs *Hintset) Hinters(ty Type) []Hinter {
	return hs.set[ty]
}

func (hs *Hintset) Compatible(ht Hint) (Hinter, error) {
	if i, err := hs.cache.Get(ht.RawString()); err == nil {
		if h, ok := i.(Hinter); ok {
			return h, nil
		}
		return nil, util.NotFoundError.Errorf("Hinter not found for %q", ht)
	} else if !errors.Is(err, gcache.KeyNotFoundError) {
		return nil, errors.Wrap(err, "Hintset cache problem")
	}

	if len(ht.Version()) < 1 {
		hinter, err := hs.Latest(ht.Type())
		if err != nil {
			_ = hs.cache.Set(ht.RawString(), err)

			return nil, err
		}

		_ = hs.cache.Set(ht.RawString(), hinter)

		return hinter, nil
	}

	hinter := hs.compatible(ht)
	if hinter == nil {
		err := util.NotFoundError.Errorf("Hinter not found for %q", ht)
		_ = hs.cache.Set(ht.RawString(), err)

		return nil, err
	}

	_ = hs.cache.Set(ht.RawString(), hinter)

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
			return errors.Errorf("empty Type, %q found", i)
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
