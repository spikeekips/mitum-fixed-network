package hint

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

// NOTE for testing only
func (st Hintset) clearCache() {
	st.cache.Range(func(k, _ interface{}) bool {
		st.cache.Delete(k)
		return true
	})
	st.unknown.Range(func(k, _ interface{}) bool {
		st.unknown.Delete(k)
		return true
	})
}

type somethingHinted struct {
	H Hint
	a int
}

func newSomethingHinted(t Type, version string, a int) somethingHinted {
	h, _ := NewHint(t, util.Version(version))
	return somethingHinted{H: h, a: a}
}

func (sh somethingHinted) Hint() Hint {
	return sh.H
}

type testHintset struct {
	suite.Suite
}

func (t *testHintset) TestNewUnRegisteredType() {
	sh := newSomethingHinted(
		Type{0xff, 0xf3},
		"2019.10-alpha",
		10,
	)
	hs := NewHintset()
	err := hs.Add(sh)
	t.True(xerrors.Is(err, UnknownTypeError))
}

func (t *testHintset) TestAdd() {
	ty := Type{0xff, 0xf1}
	t.NoError(registerType(ty, "findme"))

	sh := newSomethingHinted(ty, "2019.10", 10)

	hs := NewHintset()
	err := hs.Add(sh)
	t.NoError(err)
}

func (t *testHintset) TestRemove() {
	ty := Type{0xff, 0xf2}
	_ = registerType(ty, "showme")

	sh := newSomethingHinted(ty, "2019.10", 10)

	hs := NewHintset()
	_ = hs.Add(sh)

	// Remove unknown
	err := hs.Remove(Type{0x00, 0x00}, sh.Hint().Version())
	t.True(xerrors.Is(err, HintNotFoundError))

	err = hs.Remove(sh.Hint().Type(), sh.Hint().Version())
	t.NoError(err)
}

func (t *testHintset) TestRemoveCached() {
	ty := Type{0xff, 0xf2}
	_ = registerType(ty, "showme")

	hs := NewHintset()

	sh := newSomethingHinted(ty, "2019.10.0", 10)
	_ = hs.Add(sh)

	// Fill cache
	_, _ = hs.Hinter(sh.Hint().Type(), "")

	{
		var cached int
		hs.cache.Range(func(k, v interface{}) bool {
			cached += 1
			return true
		})
		t.Equal(1, cached)
	}

	err := hs.Remove(sh.Hint().Type(), sh.Hint().Version())
	t.NoError(err)

	var cached int
	hs.cache.Range(func(k, v interface{}) bool {
		cached += 1
		return true
	})
	t.Equal(0, cached)
}

func (t *testHintset) TestGetHint() {
	ty := Type{0xff, 0xf2}
	_ = registerType(ty, "showme")

	sh := newSomethingHinted(ty, "2019.10", 10)

	hs := NewHintset()
	_ = hs.Add(sh)

	h, err := hs.Hinter(sh.Hint().Type(), sh.Hint().Version())
	t.NoError(err)
	t.Equal(sh, h)
}

func (t *testHintset) TestGetHintWithEmptyVersion() {
	ty := Type{0xff, 0xf2}
	_ = registerType(ty, "showme")

	hs := NewHintset()

	sh := newSomethingHinted(ty, "2019.10.0", 10)
	_ = hs.Add(sh)

	{
		h, err := hs.Hinter(sh.Hint().Type(), "")
		t.NoError(err)
		t.Equal(sh, h)
	}

	hs.clearCache()

	// Register multiple, which have same type
	var latest somethingHinted
	for i := 0; i < 10; i++ {
		sh := newSomethingHinted(
			ty,
			fmt.Sprintf("2019.10.%d", i),
			10,
		)
		_ = hs.Add(sh)
		latest = sh
	}

	// will return the latest version
	h, err := hs.Hinter(sh.Hint().Type(), "")
	t.NoError(err)
	t.Equal(latest, h)
}

func TestHintset(t *testing.T) {
	suite.Run(t, new(testHintset))
}
