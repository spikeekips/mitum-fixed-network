package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type testLogging struct {
	suite.Suite
	l   *zerolog.Logger
	buf *bytes.Buffer
}

func (t *testLogging) SetupTest() {
	t.buf = bytes.NewBuffer(nil)

	l := zerolog.
		New(t.buf).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	t.l = &l
}

func (t *testLogging) TestNew() {
	logger := NewLogger(t.l, false)

	logging := NewLogging(nil)
	_ = logging.SetLogger(logger)

	logging.Log().Debug().Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("showme", m["message"])
}

func (t *testLogging) TestNilLog() {
	logging := NewLogging(nil)

	logging.Log().Verbose().Msg("showme")
	logging.Log().Debug().Msg("showme")

	t.Empty(t.buf.Bytes())
}

func (t *testLogging) TestContext() {
	logger := NewLogger(t.l, false)

	logging := NewLogging(func(ctx Context) Emitter {
		return ctx.Int("findme", 33)
	})

	_ = logging.SetLogger(logger)

	logging.Log().Debug().Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("showme", m["message"])
	t.Equal(float64(33), m["findme"])
}

func (t *testLogging) TestContextHandOver() {
	logger := NewLogger(t.l, false)

	oldLogging := NewLogging(func(ctx Context) Emitter {
		return ctx.Int("findme", 33)
	})
	_ = oldLogging.SetLogger(logger)

	logging := NewLogging(nil)
	_ = logging.SetLogger(oldLogging.Log())

	logging.Log().Debug().Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("showme", m["message"])

	// context of oldLogging will be not inherited to new logging
	_, found := m["findme"]
	t.False(found)
}

func (t *testLogging) TestContextHandOverOverride() {
	logger := NewLogger(t.l, false)

	oldLogging := NewLogging(func(ctx Context) Emitter {
		return ctx.Int("findme", 33)
	})
	_ = oldLogging.SetLogger(logger)

	logging := NewLogging(func(ctx Context) Emitter {
		return ctx.Int("findme", 44)
	})

	_ = logging.SetLogger(oldLogging.Log())

	logging.Log().Debug().Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("showme", m["message"])
	t.Equal(float64(44), m["findme"])
}

func (t *testLogging) TestNotVerbose() {
	logger := NewLogger(t.l, false)

	logging := NewLogging(func(ctx Context) Emitter {
		return ctx.Int("findme", 33)
	})

	_ = logging.SetLogger(logger)

	logging.Log().Verbose().Msg("showme")
	t.Empty(t.buf.Bytes())
}

func (t *testLogging) TestVerbose() {
	logger := NewLogger(t.l, true)

	logging := NewLogging(nil)

	_ = logging.SetLogger(logger)

	logging.Log().Verbose().Msg("showme")
	t.NotEmpty(t.buf.Bytes())

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))
	t.Equal(true, m["verbose"])
}

func (t *testLogging) TestVerboseFunc() {
	{ // not verbose
		logger := NewLogger(t.l, false)

		logging := NewLogging(nil)

		_ = logging.SetLogger(logger)

		var called bool
		logging.Log().VerboseFunc(func(e *Event) Emitter {
			called = true

			return e.Int("eatme", 44)
		}).Msg("showme")

		t.False(called)
		t.NotContains(string(t.buf.Bytes()), "eatme")
	}

	t.buf.Reset()

	{ // verbose
		logger := NewLogger(t.l, true)

		logging := NewLogging(nil)

		_ = logging.SetLogger(logger)

		var called bool
		logging.Log().VerboseFunc(func(e *Event) Emitter {
			called = true
			return e.Int("eatme", 44)
		}).Msg("showme")

		t.True(called)

		var m map[string]interface{}
		t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

		t.Equal(float64(44), m["eatme"])
	}
}

type simpleHintedLogObject struct {
	a string
}

func (h simpleHintedLogObject) MarshalLog(key string, e Emitter, _ bool) Emitter {
	return e.Str(key, h.a)
}

type hintedLogObject struct {
	a string
	b int
}

func (h hintedLogObject) MarshalLog(key string, e Emitter, verbose bool) Emitter {
	if verbose {
		return e.Dict(key, Dict().Str("a", h.a).Int("b", h.b+1)) // in verbose, b will be b + 1
	} else {
		return e.Dict(key, Dict().Str("a", h.a).Int("b", h.b))
	}
}

func (t *testLogging) TestHintedObjectSimple() {
	logger := NewLogger(t.l, true)

	logging := NewLogging(nil)

	_ = logging.SetLogger(logger)

	logging.Log().Debug().Hinted("findme", simpleHintedLogObject{a: "eatme"}).Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("eatme", m["findme"])
}

func (t *testLogging) TestHintedObject() {
	logger := NewLogger(t.l, true)

	logging := NewLogging(nil)

	_ = logging.SetLogger(logger)

	logging.Log().Debug().Hinted("findme", hintedLogObject{a: "eatme", b: 33}).Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("eatme", m["findme"].(map[string]interface{})["a"])
	t.Equal(float64(33), m["findme"].(map[string]interface{})["b"])
}

func (t *testLogging) TestHintedObjectVerbose() {
	logger := NewLogger(t.l, true)

	logging := NewLogging(nil)

	_ = logging.SetLogger(logger)

	logging.Log().Debug().HintedVerbose("findme", hintedLogObject{a: "eatme", b: 33}, true).Msg("showme")

	var m map[string]interface{}
	t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

	t.Equal("eatme", m["findme"].(map[string]interface{})["a"])
	t.Equal(float64(33+1), m["findme"].(map[string]interface{})["b"])
}

func TestLogging(t *testing.T) {
	suite.Run(t, new(testLogging))
}
