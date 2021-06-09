package hint

import (
	"fmt"
	"sort"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"
)

type hinter struct {
	h Hint
}

func newHinter(t Type, v string) hinter {
	return hinter{h: NewHint(t, v)}
}

func (sh hinter) Hint() Hint {
	return sh.h
}

type testHintset struct {
	suite.Suite
}

func (t *testHintset) TestAdd() {
	hs := NewHintset()

	ht := newHinter(Type("showme"), "v2019.10")
	t.NoError(hs.Add(ht))

	ht = newHinter(Type("findme"), "v2019.10")
	t.NoError(hs.Add(ht))
}

func (t *testHintset) TestDuplicated() {
	hs := NewHintset()

	ty := Type("showme")
	ht := newHinter(ty, "v2019.10")
	t.NoError(hs.Add(ht))
	err := hs.Add(ht)
	t.True(xerrors.Is(err, util.FoundError))
}

func (t *testHintset) TestGet() {
	hs := NewHintset()

	ty := Type("showme")
	ht := newHinter(ty, "v2019.10")
	t.NoError(hs.Add(ht))

	uht := hs.Get(ht.Hint())
	t.NotNil(uht)
	t.True(ht.Hint().Equal(uht.Hint()))
}

func (t *testHintset) TestHinters() {
	hs := NewHintset()

	{
		ty := Type("showme")

		for _, i := range []int{0, 3, 5, 10, 22, 33} {
			ht := newHinter(ty, fmt.Sprintf("v2019.10.%d", i))
			t.NoError(hs.Add(ht))
		}
	}

	ty := Type("findme")

	var hinters []Hinter
	for _, i := range []int{33, 6, 5, 22, 12, 0} {
		ht := newHinter(ty, fmt.Sprintf("v2019.10.%d", i))
		t.NoError(hs.Add(ht))

		hinters = append(hinters, ht)
	}

	sort.SliceStable(hinters, func(i, j int) bool {
		return semver.Compare(hinters[i].Hint().Version(), hinters[j].Hint().Version()) > 0
	})

	uhinters := hs.Hinters(ty)

	for i := range hinters {
		a := hinters[i]
		b := uhinters[i]

		t.True(a.Hint().Equal(b.Hint()))
	}
}

func (t *testHintset) TestLatest() {
	hs := NewHintset()

	{
		ty := Type("showme")

		for _, i := range []int{0, 3, 5, 10, 22, 33} {
			ht := newHinter(ty, fmt.Sprintf("v2019.10.%d", i))
			t.NoError(hs.Add(ht))
		}
	}

	ty := Type("findme")

	for _, i := range []int{0, 3, 5, 10, 22, 33} {
		ht := newHinter(ty, fmt.Sprintf("v2019.10.%d", i))
		t.NoError(hs.Add(ht))
	}

	i, err := hs.Latest(ty)
	t.NoError(err)
	t.True(NewHint(ty, "v2019.10.33").Equal(i.Hint()))
}

func (t *testHintset) TestCompatiblePatch() {
	hs := NewHintset()

	ty := Type("showme")

	for _, i := range []int{0, 3, 5, 10, 22, 33} {
		ht := newHinter(ty, fmt.Sprintf("v2019.10.%d", i))
		t.NoError(hs.Add(ht))
	}

	uht, err := hs.Compatible(NewHint(ty, "v2019.10.34"))
	t.NoError(err)
	t.NotNil(uht)
	t.True(NewHint(ty, "v2019.10.33").Equal(uht.Hint()))

	uht, err = hs.Compatible(NewHint(ty, "v2019.10.32"))
	t.NoError(err)
	t.NotNil(uht)
	t.True(NewHint(ty, "v2019.10.33").Equal(uht.Hint()))
}

func (t *testHintset) TestCompatibleMinor() {
	hs := NewHintset()

	ty := Type("showme")

	for _, i := range []int{0, 3, 5, 10, 22, 33} {
		ht := newHinter(ty, fmt.Sprintf("v2019.%d", i))
		t.NoError(hs.Add(ht))
	}

	uht, err := hs.Compatible(NewHint(ty, "v2019.10.34"))
	t.NoError(err)
	t.NotNil(uht)
	t.True(NewHint(ty, "v2019.33").Equal(uht.Hint()))

	uht, err = hs.Compatible(NewHint(ty, "v2019.4.32"))
	t.NoError(err)
	t.NotNil(uht)
	t.True(NewHint(ty, "v2019.33").Equal(uht.Hint()))
}

func (t *testHintset) TestCompatibleMajor() {
	hs := NewHintset()

	ty := Type("showme")

	for _, i := range []int{0, 3, 5, 10, 22, 33} {
		ht := newHinter(ty, fmt.Sprintf("v2019%d.1", i))
		t.NoError(hs.Add(ht))
	}

	var uht Hinter

	// major matched, but lower minor
	uht, err := hs.Compatible(NewHint(ty, "v201910.0"))
	t.NoError(err)
	t.NotNil(uht)
	t.True(NewHint(ty, "v201910.1").Equal(uht.Hint()))

	// major matched, but upper minor
	uht, err = hs.Compatible(NewHint(ty, "v201910.2"))
	t.True(xerrors.Is(err, util.NotFoundError))
	t.Nil(uht)

	// upper major
	uht, err = hs.Compatible(NewHint(ty, "v201934"))
	t.True(xerrors.Is(err, util.NotFoundError))
	t.Nil(uht)

	// again; get from cached
	uht, err = hs.Compatible(NewHint(ty, "v201934"))
	t.True(xerrors.Is(err, util.NotFoundError))
	t.Nil(uht)
}

func TestHintset(t *testing.T) {
	suite.Run(t, new(testHintset))
}

type testGlobalHintset struct {
	suite.Suite
}

func (t *testGlobalHintset) TestAddType() {
	hs := NewGlobalHintset()

	ht := newHinter(Type("showme"), "v2019.10")
	err := hs.Add(ht)
	t.True(xerrors.Is(err, util.NotFoundError))

	t.False(hs.HasType(ht.Hint().Type()))
	t.NoError(hs.AddType(ht.Hint().Type()))
	t.NoError(hs.Add(ht))
}

func (t *testGlobalHintset) TestInitialize() {
	hs := NewGlobalHintset()

	ht := newHinter(Type("showme"), "v2019.10")
	err := hs.Add(ht)
	t.True(xerrors.Is(err, util.NotFoundError))

	t.NoError(hs.AddType(ht.Hint().Type()))

	err = hs.Initialize()
	t.Contains(err.Error(), "empty Type")

	t.NoError(hs.Add(ht))
	t.NoError(hs.Initialize())
}

func TestGlobalHintset(t *testing.T) {
	suite.Run(t, new(testGlobalHintset))
}
