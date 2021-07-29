package logging

import (
	"bytes"

	"github.com/rs/zerolog"
)

type ZerologSTDLoggingWriter struct {
	f func() *zerolog.Event
}

func NewZerologSTDLoggingWriter(f func() *zerolog.Event) ZerologSTDLoggingWriter {
	return ZerologSTDLoggingWriter{f: f}
}

func (w ZerologSTDLoggingWriter) Write(b []byte) (int, error) {
	if w.f != nil {
		w.f().Msg(string(bytes.TrimRight(b, "\n")))
	}

	return len(b), nil
}
