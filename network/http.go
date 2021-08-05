package network

import (
	"net/http"
	"strings"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type HTTPHandlerFunc func(http.ResponseWriter, *http.Request)

func HTTPLogHandler(handler http.Handler, logger *zerolog.Logger) http.Handler {
	c := alice.New().
		Append(hlog.NewHandler(*logger)).
		Append(hlog.RemoteAddrHandler("ip")).
		Append(hlog.UserAgentHandler("user_agent")).
		Append(hlog.RefererHandler("referer")).
		Append(hlog.RequestIDHandler("req_id", "Request-Id")).
		Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			header := map[string]interface{}{}
			for k, v := range r.Header {
				switch k {
				case "Content-Length", "Content-Type", "Accept",
					"Accept-Encoding", "User-Agent":
					continue
				}

				header[k] = v
			}

			url := r.RequestURI
			if r.Method == "CONNECT" {
				url = r.Host
			}
			if url == "" {
				url = r.URL.RequestURI()
			}

			logEvent := hlog.FromRequest(r).Debug().
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Int64("content-length", r.ContentLength).
				Str("content-type", r.Header.Get("Content-Type")).
				Dict("headers", zerolog.Dict().Fields(header)).
				Str("host", r.Host).
				Str("method", r.Method).
				Str("proto", r.Proto).
				Str("remote", r.RemoteAddr).
				Str("url", url)

			if s := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); len(s) > 0 {
				logEvent = logEvent.Strs("x-forwarded-for", strings.Split(s, ","))
			}

			logEvent.Msg("request")
		}))

	return c.Then(handler)
}

func HTTPError(w http.ResponseWriter, statusCode int) {
	text := http.StatusText(statusCode)
	if len(text) < 1 {
		statusCode = http.StatusInternalServerError
		text = http.StatusText(statusCode)
	}

	http.Error(w, text, statusCode)
}
