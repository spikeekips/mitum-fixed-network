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
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Channel struct {
	*logging.Logging
	recvChan         chan seal.Seal
	connInfo         network.ConnInfo
	encs             *encoder.Encoders
	enc              encoder.Encoder
	sendSealURL      string
	getSealsURL      string
	getProposalURL   url.URL
	nodeInfoURL      string
	getBlockDataMaps string
	getBlockData     url.URL
	startHandover    string
	pingHandover     string
	endHandover      string
	client           *QuicClient
}

func NewChannel(
	connInfo network.ConnInfo,
	bufsize uint,
	quicConfig *quic.Config,
	encs *encoder.Encoders,
	enc encoder.Encoder,
) (*Channel, error) {
	ch := &Channel{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "quic-network-channel")
		}),
		recvChan: make(chan seal.Seal, bufsize),
		connInfo: connInfo,
		encs:     encs,
		enc:      enc,
	}

	addr := connInfo.URL().String()
	ch.nodeInfoURL, _ = mustQuicURL(addr, QuicHandlerPathNodeInfo)
	ch.sendSealURL, _ = mustQuicURL(addr, QuicHandlerPathSendSeal)
	ch.getSealsURL, _ = mustQuicURL(addr, QuicHandlerPathGetSeals)
	{
		_, u := mustQuicURL(addr, QuicHandlerPathGetProposal)
		ch.getProposalURL = *u
	}
	ch.getBlockDataMaps, _ = mustQuicURL(addr, QuicHandlerPathGetBlockDataMaps)
	{
		_, u := mustQuicURL(addr, QuicHandlerPathGetBlockData)
		ch.getBlockData = *u
	}
	ch.startHandover, _ = mustQuicURL(addr, QuicHandlerPathStartHandoverPattern)
	ch.pingHandover, _ = mustQuicURL(addr, QuicHandlerPathPingHandoverPattern)
	ch.endHandover, _ = mustQuicURL(addr, QuicHandlerPathEndHandoverPattern)

	client, err := NewQuicClient(connInfo.Insecure(), quicConfig)
	if err != nil {
		return nil, err
	}
	ch.client = client

	return ch, nil
}

func (*Channel) Initialize() error {
	return nil
}

func (ch *Channel) SetLogging(l *logging.Logging) *logging.Logging {
	_ = ch.client.SetLogging(l)

	return ch.Logging.SetLogging(l)
}

func (ch *Channel) ConnInfo() network.ConnInfo {
	return ch.connInfo
}

func (ch *Channel) Seals(ctx context.Context, hs []valuehash.Hash) ([]seal.Seal, error) {
	timeout := network.ChannelTimeoutSeal * time.Duration(len(hs))
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	ch.Log().Trace().Func(func(e *zerolog.Event) {
		var l []string
		for _, h := range hs {
			l = append(l, h.String())
		}

		e.Strs("seal_hashes", l)
	}).Msg("request seals")

	ss, err := ch.doRequestHinters(ctx, ch.client.Send, timeout+(time.Second*2), ch.getSealsURL, NewHashesArgs(hs))
	if err != nil {
		return nil, err
	}

	seals := make([]seal.Seal, len(ss))
	for i := range ss {
		h := ss[i]
		s, ok := h.(seal.Seal)
		if !ok {
			return nil, errors.Errorf("decoded, but not seal.Seal; %T", h)
		}
		seals[i] = s
	}

	return seals, nil
}

func (ch *Channel) SendSeal(ctx context.Context, ci network.ConnInfo, sl seal.Seal) error {
	l := ch.Log().With().Stringer("cid", util.UUID()).Stringer("seal_hash", sl.Hash()).Logger()

	l.Trace().Msg("trying to send seal")

	timeout := network.ChannelTimeoutSendSeal
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	b, err := ch.enc.Marshal(sl)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())
	if ci != nil {
		headers.Set(SendSealFromConnInfoHeader, ci.String())
	}

	res, err := ch.client.Send(ctx, timeout*2, ch.sendSealURL, b, headers)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Close()

		l.Trace().Msg("seal sent")
	}()

	return nil
}

func (ch *Channel) Proposal(ctx context.Context, h valuehash.Hash) (base.Proposal, error) {
	ctx, cancel := ch.timeoutContext(ctx, network.ChannelTimeoutSeal)
	defer cancel()

	ch.Log().Trace().Stringer("proposal", h).Msg("request proposal")

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	u := ch.getProposalURL
	u.Path = u.Path + "/" + h.String()

	response, err := ch.client.Get(ctx, network.ChannelTimeoutSeal, u.String(), nil, headers)
	defer func() {
		if response == nil {
			return
		}

		_ = response.Close()
	}()

	if err != nil {
		return nil, err
	} else if err = response.Error(); err != nil {
		return nil, err
	}

	enc, err := EncoderFromHeader(response.Header, ch.encs, ch.enc)
	if err != nil {
		return nil, err
	}

	b, err := response.Bytes()
	if err != nil {
		ch.Log().Error().Err(err).Msg("failed to get bytes from response body")

		return nil, err
	}

	return base.DecodeProposal(b, enc)
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
	} else if err = response.Error(); err != nil {
		return nil, err
	}

	enc, err := EncoderFromHeader(response.Header, ch.encs, ch.enc)
	if err != nil {
		return nil, err
	}

	if b, err := response.Bytes(); err != nil {
		ch.Log().Error().Err(err).Msg("failed to get bytes from response body")

		return nil, err
	} else if i, err := network.DecodeNodeInfo(b, enc); err != nil {
		return nil, err
	} else {
		return i, nil
	}
}

func (ch *Channel) BlockDataMaps(ctx context.Context, heights []base.Height) ([]block.BlockDataMap, error) {
	timeout := network.ChannelTimeoutBlockDataMap * time.Duration(len(heights))
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	ch.Log().Trace().Func(func(e *zerolog.Event) {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		e.Strs("heights", l)
	}).Msg("request block data maps")

	hinters, err := ch.doRequestHinters(
		ctx,
		ch.client.Send,
		timeout+(time.Second*2),
		ch.getBlockDataMaps, NewHeightsArgs(heights),
	)
	if err != nil {
		return nil, err
	}

	var bds []block.BlockDataMap
	for _, h := range hinters {
		if s, ok := h.(block.BlockDataMap); !ok {
			return nil, errors.Errorf("decoded, but not BlockDataMap; %T", h)
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

func (ch *Channel) StartHandover(ctx context.Context, sl network.StartHandoverSeal) (bool, error) {
	return ch.sendHandoverSeal(ctx, ch.startHandover, sl)
}

func (ch *Channel) PingHandover(ctx context.Context, sl network.PingHandoverSeal) (bool, error) {
	return ch.sendHandoverSeal(ctx, ch.pingHandover, sl)
}

func (ch *Channel) EndHandover(ctx context.Context, sl network.EndHandoverSeal) (bool, error) {
	return ch.sendHandoverSeal(ctx, ch.endHandover, sl)
}

func (ch *Channel) blockData(ctx context.Context, p string) (io.ReadCloser, func() error, error) {
	ch.Log().Trace().Str("path", p).Msg("request block data")

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
	ctx context.Context,
	f clientDoRequestFunc,
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
	} else if err = response.Error(); err != nil {
		return nil, err
	}

	enc, err := EncoderFromHeader(response.Header, ch.encs, ch.enc)
	if err != nil {
		return nil, err
	}

	var ss []json.RawMessage
	if b, err := response.Bytes(); err != nil {
		ch.Log().Error().Err(err).Msg("failed to get bytes from response body")

		return nil, err
	} else if err := enc.Unmarshal(b, &ss); err != nil {
		ch.Log().Error().Err(err).Msg("failed to unmarshal manifest slice")
		return nil, err
	}

	hinters := make([]hint.Hinter, len(ss))
	for i := range ss {
		hinter, err := enc.Decode(ss[i])
		if err != nil {
			return nil, err
		}
		hinters[i] = hinter
	}

	return hinters, nil
}

func (*Channel) timeoutContext(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	switch {
	case ctx != context.TODO():
		return ctx, func() {}
	case timeout < 1:
		return ctx, func() {}
	}

	return context.WithTimeout(context.Background(), timeout)
}

func (ch *Channel) sendHandoverSeal(ctx context.Context, path string, sl network.HandoverSeal) (bool, error) {
	timeout := network.ChannelTimeoutHandover
	ctx, cancel := ch.timeoutContext(ctx, timeout)
	defer cancel()

	b, err := ch.enc.Marshal(sl)
	if err != nil {
		return false, err
	}

	l := ch.Log().With().Stringer("seal_hash", sl.Hash()).Stringer("hint", sl.Hint()).Logger()

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	res, err := ch.client.Send(ctx, 0 /* set to default, 30s */, path, b, headers)
	if err != nil {
		l.Error().Err(err).Msg("failed to send handover seal")

		return false, err
	}

	defer func() {
		_ = res.Close()
	}()

	e := l.Trace().Int("status_code", res.StatusCode)

	switch {
	case res.StatusCode == http.StatusOK, res.StatusCode == http.StatusCreated:
		e.Msg("successfully sent handover seal")

		return true, nil
	case network.IsProblemFromResponse(res.Response):
		problem, err := network.LoadProblemFromResponse(res.Response)
		if err != nil {
			e.Err(err).Msg("sent handover seal")

			return false, err
		}

		e.Interface("problem", problem).Msg("sent handover seal")

		return false, problem
	default:
		e.Msg("sent handover seal, but said no")

		return false, nil
	}
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
