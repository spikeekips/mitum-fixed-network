package condition

import (
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
)

type LogItem struct {
	bytes []byte
	m     map[string]interface{}
}

func NewLogItem(b []byte) (LogItem, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return LogItem{}, err
	}

	return LogItem{bytes: b, m: m}, nil
}

func NewLogItemFromMap(m map[string]interface{}) (LogItem, error) {
	return LogItem{m: m}, nil
}

func (li LogItem) Bytes() []byte {
	if li.bytes == nil {
		b, err := json.Marshal(li.m)
		if err != nil {
			return nil
		}
		li.bytes = b
	}

	return li.bytes
}

func (li LogItem) Map() map[string]interface{} {
	return li.m
}

type LogWatcher struct {
	sync.RWMutex
	*common.Logger
	q             *queue.Queue
	cc            *MultipleConditionChecker
	acc           []ActionChecker
	satisfiedChan chan bool
	stopped       bool
}

func NewLogWatcher(
	cc *MultipleConditionChecker,
	satisfiedChan chan bool,
) *LogWatcher {
	return &LogWatcher{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "log-watcher")
		}),
		q:             queue.New(100000),
		cc:            cc,
		satisfiedChan: satisfiedChan,
	}
}

func (lw *LogWatcher) SetActionCheckers(acc []ActionChecker) {
	lw.acc = acc
}

func (lw *LogWatcher) Stop() error {
	lw.Lock()
	lw.stopped = true
	lw.Unlock()

	// empty queue
	for {
		if lw.q.Len() < 1 {
			break
		}

		_, _ = lw.q.Get(10000)
	}

	return nil
}

func (lw *LogWatcher) Write(b []byte) (int, error) {
	lw.RLock()
	if lw.stopped {
		lw.RUnlock()
		return len(b), nil
	}
	lw.RUnlock()

	if err := lw.q.Put(string(b)); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (lw *LogWatcher) peek() (string, error) {
	b, err := lw.q.Peek()
	if err != nil {
		return "", err
	}

	return b.(string), nil
}

func (lw *LogWatcher) Left() bool {
	return lw.q.Len() > 0
}

func (lw *LogWatcher) Start() error {
	check := func() bool {
		if lw.q.Len() < 1 {
			<-time.After(time.Millisecond * 100)
			return false
		}

		defer func() {
			if _, err := lw.q.Get(1); err != nil {
				lw.Log().Error().Err(err).
					Msg("failed to get log item")
			}
		}()

		b, err := lw.peek()
		if err != nil {
			lw.Log().Error().Err(err).
				Msg("failed to peek log")
			return false
		} else if len(b) < 1 {
			return false
		}

		o, err := NewLogItem([]byte(b))
		if err != nil {
			lw.Log().Error().Err(err).
				Msg("failed to make LogItem")
			return false
		}

		return lw.check(o)
	}

	go func() {
		for {
			if check() {
				break
			}
		}

		_ = lw.Stop()

		lw.satisfiedChan <- true
	}()

	return nil
}

func (lw *LogWatcher) CheckBytes(b []byte) bool {
	o, err := NewLogItem(b)
	if err != nil {
		lw.Log().Error().Err(err).
			Msg("failed to make LogItem")
		return false
	}

	return lw.check(o)
}

func (lw *LogWatcher) check(o LogItem) bool {
	if lw.acc != nil {
		for _, c := range lw.acc {
			if !c.Checker().Check(o) {
				continue
			}
			for _, action := range c.Actions() {
				if len(action.Value().Value()) < 1 {
					continue
				} else if action.Value().Hint() != reflect.Func {
					continue
				}
				go action.Value().Value()[0].(func())()
			}
		}
	}

	if lw.cc != nil && lw.cc.Check(o) && lw.cc.AllSatisfied() {
		return true
	}

	return false
}

type HighlightWriter struct {
	w         io.Writer
	lexer     chroma.Lexer
	formatter chroma.Formatter
	style     *chroma.Style
}

func NewHighlightWriter(w io.Writer) HighlightWriter {
	lexer := "json"
	formatter := "terminal16m"
	// style := "github"
	// style := "monokai"
	// style := "vim"
	style := "native"

	f := formatters.Get(formatter)
	if f == nil {
		f = formatters.Fallback
	}

	s := styles.Get(style)
	if s == nil {
		s = styles.Fallback
	}

	l := chroma.Coalesce(lexers.Get(lexer))

	return HighlightWriter{w: w, formatter: f, lexer: l, style: s}
}

func (hw HighlightWriter) Write(b []byte) (int, error) {
	it, err := hw.lexer.Tokenise(nil, string(b))
	if err != nil {
		return 0, err
	}

	if err = hw.formatter.Format(hw.w, hw.style, it); err != nil {
		return 0, err
	}

	return len(b), nil
}
