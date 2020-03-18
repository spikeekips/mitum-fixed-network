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

	logging := NewLogging(func(ctx zerolog.Context) zerolog.Context {
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

	oldLogging := NewLogging(func(ctx zerolog.Context) zerolog.Context {
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

	oldLogging := NewLogging(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Int("findme", 33)
	})
	_ = oldLogging.SetLogger(logger)

	logging := NewLogging(func(ctx zerolog.Context) zerolog.Context {
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

	logging := NewLogging(func(ctx zerolog.Context) zerolog.Context {
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
		logging.Log().VerboseFunc(func(e *zerolog.Event) *zerolog.Event {
			called = true

			return e.Int("eatme", 44)
		}).Msg("showme")

		t.False(called)
		t.Empty(t.buf.Bytes())
	}

	{ // verbose
		logger := NewLogger(t.l, true)

		logging := NewLogging(nil)

		_ = logging.SetLogger(logger)

		var called bool
		logging.Log().VerboseFunc(func(e *zerolog.Event) *zerolog.Event {
			called = true
			return e.Int("eatme", 44)
		}).Msg("showme")

		t.True(called)

		var m map[string]interface{}
		t.NoError(json.Unmarshal(t.buf.Bytes(), &m))

		t.Equal(float64(44), m["eatme"])
	}
}

func TestLogging(t *testing.T) {
	suite.Run(t, new(testLogging))
}
