package quicnetwork

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lucas-clemente/quic-go"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Channel struct {
	*logging.Logging
	recvChan         chan seal.Seal
	u                string
	addr             *url.URL
	encs             *encoder.Encoders
	enc              encoder.Encoder
	sendSealURL      string
	getSealsURL      string
	nodeInfoURL      string
	getBlockDataMaps string
	getBlockData     url.URL
	client           *QuicClient
}

func NewChannel(
	addr string,
	bufsize uint,
	quicConfig *quic.Config,
	encs *encoder.Encoders,
	enc encoder.Encoder,
) (*Channel, error) {
	ch := &Channel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "quic-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
		encs:     encs,
		enc:      enc,
	}

	var insecure bool
	if u, err := url.Parse(addr); err != nil {
		return nil, err
	} else {
		if u.Scheme == "quic" {
			u.Scheme = "https"
		}

		query := u.Query()
		insecure = parseBoolInQuery(u.Query().Get("insecure"))
		query.Del("insecure")
		u.RawQuery = query.Encode()

		ch.addr = u
		ch.u = addr
	}

	ch.nodeInfoURL, _ = mustQuicURL(ch.addr.String(), QuicHandlerPathNodeInfo)
	ch.sendSealURL, _ = mustQuicURL(ch.addr.String(), QuicHandlerPathSendSeal)
	ch.getSealsURL, _ = mustQuicURL(ch.addr.String(), QuicHandlerPathGetSeals)
	ch.getBlockDataMaps, _ = mustQuicURL(ch.addr.String(), QuicHandlerPathGetBlockDataMaps)
	{
		_, u := mustQuicURL(ch.addr.String(), QuicHandlerPathGetBlockData)
		ch.getBlockData = *u
	}

	if client, err := NewQuicClient(insecure, quicConfig); err != nil {
		return nil, err
	} else {
		ch.client = client
	}

	return ch, nil
}

func (ch *Channel) Initialize() error {
	return nil
}

func (ch *Channel) SetLogger(l logging.Logger) logging.Logger {
	_ = ch.Logging.SetLogger(l)
	_ = ch.client.SetLogger(l)

	return ch.Log()
}

func (ch *Channel) URL() string {
	return ch.u
}

func (ch *Channel) Seals(ctx context.Context, hs []valuehash.Hash) ([]seal.Seal, error) {
	timeout := network.ChannelTimeoutSeal * time.Duration(len(hs))
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range hs {
			l = append(l, h.String())
		}

		return e.Strs("seal_hashes", l)
	}).Msg("request seals")

	ss, err := ch.doRequestHinters(ch.client.Send, ctx, timeout+(time.Second*2), ch.getSealsURL, NewHashesArgs(hs))
	if err != nil {
		return nil, err
	}

	var seals []seal.Seal
	for _, h := range ss {
		if s, ok := h.(seal.Seal); !ok {
			return nil, xerrors.Errorf("decoded, but not seal.Seal; %T", h)
		} else {
			seals = append(seals, s)
		}
	}

	return seals, nil
}

func (ch *Channel) SendSeal(ctx context.Context, sl seal.Seal) error {
	timeout := network.ChannelTimeoutSendSeal
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	b, err := ch.enc.Marshal(sl)
	if err != nil {
		return err
	}

	ch.Log().Debug().Hinted("seal_hash", sl.Hash()).Msg("sent seal")

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	if res, err := ch.client.Send(ctx, timeout*2, ch.sendSealURL, b, headers); err != nil {
		return err
	} else {
		defer func() {
			_ = res.Close()
		}()

		return nil
	}
}

func (ch *Channel) NodeInfo(ctx context.Context) (network.NodeInfo, error) {
	timeout := network.ChannelTimeoutNodeInfo
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	response, err := ch.client.Get(ctx, timeout*2, ch.nodeInfoURL, nil, headers)
	defer func() {
		if response == nil {
			return
		}

		_ = response.Close()
	}()

	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header, ch.encs, ch.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	if b, err := response.Bytes(); err != nil {
		ch.Log().Error().Err(err).Msg("failed to get bytes from response body")

		return nil, err
	} else if hinter, err := network.DecodeNodeInfo(enc, b); err != nil {
		return nil, err
	} else {
		return hinter.(network.NodeInfo), nil
	}
}

func (ch *Channel) BlockDataMaps(ctx context.Context, heights []base.Height) ([]block.BlockDataMap, error) {
	timeout := network.ChannelTimeoutBlockDataMap * time.Duration(len(heights))
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		return e.Strs("heights", l)
	}).Msg("request block data maps")

	hinters, err := ch.doRequestHinters(
		ch.client.Send,
		ctx, timeout+(time.Second*2),
		ch.getBlockDataMaps, NewHeightsArgs(heights),
	)
	if err != nil {
		return nil, err
	}

	var bds []block.BlockDataMap
	for _, h := range hinters {
		if s, ok := h.(block.BlockDataMap); !ok {
			return nil, xerrors.Errorf("decoded, but not BlockDataMap; %T", h)
		} else if err := s.IsValid(nil); err != nil {
			return nil, isvalid.InvalidError.Errorf("invalid block data map: %w", err)
		} else {
			bds = append(bds, s)
		}
	}

	return bds, nil
}

func (ch *Channel) BlockData(ctx context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	ctx, cancel := ch.timeoutContext(ctx, network.ChannelTimeoutBlockData)
	defer cancel()

	return network.FetchBlockDataThruChannel(
		func(p string) (io.Reader, func() error, error) {
			return ch.blockData(ctx, p)
		},
		item,
	)
}

func (ch *Channel) blockData(ctx context.Context, p string) (io.ReadCloser, func() error, error) {
	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		return e.Str("path", p)
	}).Msg("request block data")

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	u := ch.getBlockData
	u.Path = u.Path + "/" + stripSlashFilePath(p)

	response, err := ch.client.Get(ctx, time.Minute, u.String(), nil, headers)
	closeFunc := func() error {
		if response == nil {
			return nil
		}

		return response.Close()
	}

	if err != nil {
		return nil, closeFunc, err
	} else if err := response.Error(); err != nil {
		return nil, closeFunc, err
	}

	return response.Body(), closeFunc, nil
}

func (ch *Channel) doRequestHinters(
	f clientDoRequestFunc,
	ctx context.Context,
	timeout time.Duration,
	u string,
	hs interface{},
) ([]hint.Hinter, error) {
	b, err := ch.enc.Marshal(hs)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	response, err := f(ctx, timeout, u, b, headers)
	defer func() {
		if response == nil {
			return
		}

		_ = response.Close()
	}()

	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header, ch.encs, ch.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	var ss []json.RawMessage
	if b, err := response.Bytes(); err != nil {
		ch.Log().Error().Err(err).Msg("failed to get bytes from response body")

		return nil, err
	} else if err := enc.Unmarshal(b, &ss); err != nil {
		ch.Log().Error().Err(err).Msg("failed to unmarshal manifest slice")
		return nil, err
	}

	var hinters []hint.Hinter
	for _, r := range ss {
		if hinter, err := enc.DecodeByHint(r); err != nil {
			return nil, err
		} else {
			hinters = append(hinters, hinter)
		}
	}

	return hinters, nil
}

func (ch *Channel) timeoutContext(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	switch {
	case ctx != context.TODO():
		return ctx, func() {}
	case timeout < 1:
		return ctx, func() {}
	}

	return context.WithTimeout(context.Background(), timeout)
}

var reStripSlash = regexp.MustCompile(`^[/]*`)

func stripSlashFilePath(p string) string {
	p = strings.TrimSpace(p)
	if len(p) < 1 {
		return ""
	}

	b := reStripSlash.ReplaceAll([]byte(p), nil)

	return string(b)
}

func parseBoolInQuery(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "t", "true", "1", "y", "yes":
		return true
	default:
		return false
	}
}
