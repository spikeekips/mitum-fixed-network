package condition

import (
	"encoding/json"
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
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

func (li LogItem) Bytes() []byte {
	return li.bytes
}

func (li LogItem) Map() map[string]interface{} {
	return li.m
}

type LogWatcher struct {
	conditionChecker *MultipleConditionChecker
	chanRead         chan LogItem
	satisfiedChan    chan bool
}

func NewLogWatcher(
	conditionChecker *MultipleConditionChecker,
	satisfiedChan chan bool,
) *LogWatcher {
	return &LogWatcher{
		conditionChecker: conditionChecker,
		chanRead:         make(chan LogItem),
		satisfiedChan:    satisfiedChan,
	}
}

func (lw *LogWatcher) Write(b []byte) {
	i, err := NewLogItem(b)
	if err != nil {
		return
	}

	lw.chanRead <- i
}

func (lw *LogWatcher) Start() error {
	go func() {
		for o := range lw.chanRead {
			if lw.check(o) {
				lw.satisfiedChan <- true
			}
		}
	}()

	return nil
}

func (lw *LogWatcher) check(o LogItem) bool {
	if lw.conditionChecker == nil {
		return false
	}

	if !lw.conditionChecker.Check(o) {
		return false
	}

	return lw.conditionChecker.AllSatisfied()
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
