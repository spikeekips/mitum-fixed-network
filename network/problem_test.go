package network

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testProblem struct {
	suite.Suite
}

func (t *testProblem) TestNew() {
	pt := "showme"
	title := "killme"
	pr := NewProblem(pt, title)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(jsonenc.Unmarshal(b, &m))

	t.Contains(m["type"], pt)
	t.Equal(title, m["title"])
	t.Empty(m["detail"])
}

func (t *testProblem) TestExtra() {
	pt := "showme"
	title := "killme"
	pr := NewProblem(pt, title)
	pr = pr.AddExtra("a", []string{"1", "2"})

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var upr Problem
	t.NoError(jsonenc.Unmarshal(b, &upr))

	t.Contains(upr.Type(), pt)
	t.Equal(title, upr.Title())
	t.Empty(upr.Detail())
	t.Equal([]interface{}{"1", "2"}, upr.Extra()["a"])
}

func (t *testProblem) TestFromError() {
	e := util.NewError("showme")
	pr := NewProblemFromError(e)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var upr Problem
	t.NoError(jsonenc.Unmarshal(b, &upr))

	t.Contains(DefaultProblemType, upr.Type())
	t.Equal("showme", upr.Title())
	t.Equal("showme", upr.Detail())
}

func (t *testProblem) TestFromWrapedError() {
	e0 := errors.Errorf("showme")
	e := errors.Wrapf(e0, "findme")
	pr := NewProblemFromError(e)

	b, err := jsonenc.Marshal(pr)
	t.NoError(err)

	var upr Problem
	t.NoError(jsonenc.Unmarshal(b, &upr))

	t.Contains(DefaultProblemType, upr.Type())
	t.Equal(upr.Title(), "findme: showme")
	t.Contains(upr.Detail(), "findme")
	t.Contains(upr.Detail(), "problem_test.go")
}

func TestProblem(t *testing.T) {
	suite.Run(t, new(testProblem))
}
