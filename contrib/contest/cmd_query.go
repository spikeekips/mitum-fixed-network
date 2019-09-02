package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/spf13/cobra"

	"github.com/spikeekips/mitum/contrib/contest/condition"
)

var queryCmd = &cobra.Command{
	Use:   "query <log>",
	Short: "query logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lf, err := os.Open(args[0])
		if err != nil {
			cmd.Println("Error: failed to open log file:", err.Error())
			os.Exit(1)
		}
		defer lf.Close() // nolint

		log.Debug().
			Str("query", flagQuery).
			Msg("query")

		var conditionChecker *condition.ConditionChecker
		if len(flagQuery) > 0 {
			cc, err := condition.NewConditionChecker(flagQuery)
			if err != nil {
				cmd.Println("Error: wrong query:", err.Error())
				os.Exit(1)
			}
			conditionChecker = &cc
		}

		lw := NewLogWatcher(conditionChecker, flagJSONPretty)
		lw.Start()

		reader := bufio.NewReader(lf)
		for {
			b, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			lw.Write(b)
		}
	},
}

func init() {
	queryCmd.Flags().StringVar(&flagQuery, "query", "", "query")
	queryCmd.Flags().BoolVar(&flagJSONPretty, "pretty", false, "pretty json output")

	rootCmd.AddCommand(queryCmd)
}

type LogItem struct {
	raw []byte
	o   map[string]interface{}
}

func NewLogItem(b []byte) (LogItem, error) {
	o := map[string]interface{}{}
	if err := json.Unmarshal(b, &o); err != nil {
		return LogItem{}, err
	}

	return LogItem{raw: b, o: o}, nil
}

type LogWatcher struct {
	conditionChecker *condition.ConditionChecker
	chanRead         chan LogItem
	enc              *json.Encoder
	hw               HighlightWriter
}

func NewLogWatcher(
	conditionChecker *condition.ConditionChecker,
	pretty bool,
) *LogWatcher {
	var enc *json.Encoder

	hw := NewHighlightWriter(os.Stdout)
	if pretty {
		enc = json.NewEncoder(hw)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
	}

	return &LogWatcher{
		conditionChecker: conditionChecker,
		chanRead:         make(chan LogItem),
		enc:              enc,
		hw:               hw,
	}
}

func (lw *LogWatcher) Write(b []byte) {
	i, err := NewLogItem(b)
	if err != nil {
		log.Debug().
			Err(err).
			Str("data", string(b)).
			Msg("invalid data found")
		return
	}

	lw.chanRead <- i
}

func (lw *LogWatcher) Start() {
	go func() {
		for o := range lw.chanRead {
			if lw.conditionChecker == nil {
				fmt.Println(o)
				continue
			}

			lw.check(o)
		}
	}()
}

func (lw *LogWatcher) check(o LogItem) {
	if !lw.conditionChecker.Check(o.o) {
		return
	}

	if lw.enc != nil {
		lw.enc.Encode(json.RawMessage(o.raw))
		return
	}
	fmt.Fprint(lw.hw, string(o.raw))
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
