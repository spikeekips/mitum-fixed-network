package cmds

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type QuicClientCommand struct {
	*BaseCommand
	URL        *url.URL      `arg:"" name:"node url" help:"remote mitum url" required:"true"`
	Timeout    time.Duration `name:"timeout" help:"timeout; default is 5 seconds"`
	TLSInscure bool          `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	Method     string        `name:"method" help:"http method {GET HEAD POST PUT DELETE CONNECT OPTIONS TRACE PATCH}; default is 'GET'"` // revive:disable-line:line-length-limit
	Headers    []string      `name:"header" help:"http header; <key>: <value>"`
	Body       FileLoad      `name:"body" help:"set http body" optional:""`
	JSON       bool          `name:"json" help:"json output format (default: false)" optional:"" default:"false"`
	headers    http.Header
}

func NewQuicClientCommand() QuicClientCommand {
	return QuicClientCommand{
		BaseCommand: NewBaseCommand("quic_client"),
	}
}

func (cmd *QuicClientCommand) Initialize(flags interface{}, version util.Version) error {
	if err := cmd.BaseCommand.Initialize(flags, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	cmd.Method = strings.TrimSpace(cmd.Method)
	if len(cmd.Method) < 1 {
		cmd.Method = "GET"
	}

	headers, err := cmd.loadHeaders()
	if err != nil {
		return err
	}
	cmd.headers = headers

	return nil
}

func (cmd *QuicClientCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	cmd.Log().Debug().
		Interface("url", cmd.URL).
		Interface("headers", cmd.headers).
		Msg("trying to request")

	quicConfig := &quic.Config{HandshakeIdleTimeout: cmd.Timeout}
	client, err := quicnetwork.NewQuicClient(cmd.TLSInscure, quicConfig)
	if err != nil {
		return fmt.Errorf("failed to create quic client: %w", err)
	}

	_ = client.SetLogging(cmd.Logging)

	res, closefunc, err := client.Request( // nolint:bodyclose
		context.Background(),
		cmd.Timeout,
		cmd.URL.String(),
		cmd.Method,
		cmd.Body.Bytes(),
		cmd.headers,
	)
	if err != nil {
		defer func() {
			_ = closefunc()
		}()

		return fmt.Errorf("failed to request: %w", err)
	}

	response := quicnetwork.NewQuicResponse(res, closefunc)
	defer func() {
		_ = response.Close()
	}()

	cmd.Log().Debug().
		Interface("response", response).
		Msg("requested")

	cmd.print(response)

	return nil
}

func (cmd *QuicClientCommand) loadHeaders() (http.Header, error) {
	if len(cmd.Headers) < 1 {
		return http.Header{}, nil
	}

	var hs string
	for i := range cmd.Headers {
		hs += fmt.Sprintf("%s\r\n", cmd.Headers[i])
	}

	tp := textproto.NewReader(bufio.NewReader(strings.NewReader(hs + "\r\n")))

	m, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to load http headers: %w", err)
	}

	return http.Header(m), nil
}

func (cmd *QuicClientCommand) print(response *quicnetwork.QuicResponse) {
	body, err := response.Bytes()
	if err != nil {
		cmd.Log().Warn().Msg("failed to read body")
	}

	var isJSONBody bool
	if strings.Contains(response.Header.Get(quicnetwork.QuicEncoderHintHeader), "json") {
		isJSONBody = true
	}

	if cmd.JSON {
		var outputbody interface{}
		if isJSONBody {
			outputbody = json.RawMessage(body)
		} else {
			outputbody = string(body)
		}

		_, _ = fmt.Fprintln(os.Stdout, string(jsonenc.MustMarshal(
			map[string]interface{}{
				"status":            response.Status,
				"status_code":       response.StatusCode,
				"headers":           response.Header,
				"proto":             response.Proto,
				"content_length":    response.ContentLength,
				"transfer_encoding": response.TransferEncoding,
				"body_length":       len(body),
				"body":              outputbody,
			},
		)))

		return
	}

	_, _ = fmt.Fprintln(os.Stdout, "> response")
	_, _ = fmt.Fprintln(os.Stdout, "           status:", response.Status)
	_, _ = fmt.Fprintln(os.Stdout, "      status_code:", response.StatusCode)
	hk := make([]string, len(response.Header))
	var i int
	for k := range response.Header {
		hk[i] = k
		i++
	}
	sort.Strings(hk)

	_, _ = fmt.Fprintln(os.Stdout, "          headers:")
	for i := range hk {
		_, _ = fmt.Fprintln(os.Stdout, "                  ", fmt.Sprintf("%s: %s", hk[i], response.Header.Get(hk[i])))
	}
	_, _ = fmt.Fprintln(os.Stdout, "            proto:", response.Proto)
	_, _ = fmt.Fprintln(os.Stdout, "   content-length:", response.ContentLength)
	_, _ = fmt.Fprintln(os.Stdout, "transfer-encoding:", response.TransferEncoding)
	_, _ = fmt.Fprintln(os.Stdout, "      body-length:", len(body))

	if !isJSONBody {
		_, _ = fmt.Fprintln(os.Stdout, "             body:", string(body))
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "             body:")
		_, _ = fmt.Fprintln(os.Stdout, string(jsonenc.MustMarshalIndent(json.RawMessage(body))))
	}
}
