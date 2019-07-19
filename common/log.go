package common

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"boscoin.io/sebak/lib/errors"
	"github.com/inconshreveable/log15"
	"github.com/mattn/go-isatty"
)

var log log15.Logger = log15.New("module", "common")

func Log() log15.Logger {
	return log
}

func LogFormatter(f string) log15.Format {
	var logFormatter log15.Format
	switch f {
	case "terminal":
		if isatty.IsTerminal(os.Stdout.Fd()) {
			logFormatter = log15.TerminalFormat()
		} else {
			logFormatter = log15.LogfmtFormat()
		}
	case "", "json":
		logFormatter = JsonFormatEx()
	}

	return logFormatter
}

func LogHandler(format log15.Format, f string) (log15.Handler, error) {
	if len(f) < 1 {
		return log15.StreamHandler(os.Stdout, format), nil
	}

	return log15.FileHandler(f, format)
}

func SetLogger(logger log15.Logger, level log15.Lvl, handler log15.Handler) {
	logger.SetHandler(log15.LvlFilterHandler(level, handler))
}

// `formatJSONValue` and `JsonFormatEx` was derived from
// https://github.com/inconshreveable/log15/blob/199fca55789248e0520a3bd33e9045799738e793/format.go#L131
// .
const errorKey = "LOG15_ERROR"

func formatLogJSONValue(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case *errors.Error:
		result = v
	case Time:
		result = v.String()
	case time.Duration:
		result = v.String()
	case time.Time:
		result = v.Format(TIMEFORMAT_ISO8601)
	case json.Marshaler:
		result = v
	case error:
		result = v.Error()
	case fmt.Stringer:
		result = v.String()
	default:
		result = v
	}

	return
}

func JsonFormatEx() log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		props := make(map[string]interface{})

		props[r.KeyNames.Time] = r.Time
		props[r.KeyNames.Lvl] = r.Lvl.String()
		props[r.KeyNames.Msg] = r.Msg

		for i := 0; i < len(r.Ctx); i += 2 {
			k, ok := r.Ctx[i].(string)
			if !ok {
				props[errorKey] = fmt.Sprintf("%+v is not a string key", r.Ctx[i])
			}
			props[k] = formatLogJSONValue(r.Ctx[i+1])
		}

		b, err := EncodeJSON(props, false, false)
		if err != nil {
			b, _ = json.Marshal(map[string]string{
				errorKey: err.Error(),
			})
			return b
		}

		b = append(b, '\n')

		return b
	})
}
