package isaac

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type QuicChannel struct {
	*logging.Logging
	recvChan    chan seal.Seal
	addr        *url.URL
	encs        *encoder.Encoders
	enc         encoder.Encoder
	sendSealURL string
	getSealsURL string
	client      *network.QuicClient
}

func NewQuicChannel(
	addr string, // TODO use "quic" scheme
	bufsize uint,
	insecure bool,
	timeout time.Duration,
	retries int,
	quicConfig *quic.Config,
	encs *encoder.Encoders,
	enc encoder.Encoder,
) (*QuicChannel, error) {
	qc := &QuicChannel{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "quic-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
		encs:     encs,
		enc:      enc,
	}

	if u, err := url.Parse(addr); err != nil {
		return nil, err
	} else {
		qc.addr = u
	}

	if u, err := QuicSendSealURL(qc.addr.String()); err != nil {
		return nil, err
	} else {
		qc.sendSealURL = u
	}
	if u, err := QuicGetSealsURL(qc.addr.String()); err != nil {
		return nil, err
	} else {
		qc.getSealsURL = u
	}

	if client, err := network.NewQuicClient(insecure, timeout, retries, quicConfig); err != nil {
		return qc, nil
	} else {
		qc.client = client
	}

	return qc, nil
}

func (qc *QuicChannel) SetLogger(l logging.Logger) logging.Logger {
	_ = qc.Logging.SetLogger(l)
	_ = qc.client.SetLogger(l)

	return qc.Log()
}

func (qc *QuicChannel) URL() *url.URL {
	return qc.addr
}

func (qc *QuicChannel) Seals(hs []valuehash.Hash) ([]seal.Seal, error) {
	b, err := qc.enc.Marshal(hs)
	if err != nil {
		return nil, err
	}

	if qc.Log().IsVerbose() {
		var l []string
		for _, h := range hs {
			l = append(l, h.String())
		}

		qc.Log().Debug().Strs("seal_hashes", l).Msg("request seals")
	}

	headers := http.Header{}
	headers.Set(network.QuicEncoderHintHeader, qc.enc.Hint().String())

	response, err := qc.client.Request(qc.getSealsURL, b, headers)
	if err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := network.EncoderFromHeader(response.Header(), qc.encs, qc.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	var ss []json.RawMessage
	if err := enc.Unmarshal(response.Bytes(), &ss); err != nil {
		qc.Log().Error().Err(err).Msg("failed to unmarshal seal slice")
		return nil, err
	}

	var seals []seal.Seal
	for _, r := range ss {
		if hinter, err := enc.DecodeByHint(r); err != nil {
			return nil, err
		} else if s, ok := hinter.(seal.Seal); !ok {
			return nil, xerrors.Errorf("decoded, but not seal.Seal; %T", hinter)
		} else {
			seals = append(seals, s)
		}
	}

	return seals, nil
}

func (qc *QuicChannel) SendSeal(sl seal.Seal) error {
	b, err := qc.enc.Marshal(sl)
	if err != nil {
		return err
	}

	qc.Log().Debug().Str("seal_hash", sl.Hash().String()).Msg("sent seal")

	headers := http.Header{}
	headers.Set(network.QuicEncoderHintHeader, qc.enc.Hint().String())

	return qc.client.Send(qc.sendSealURL, b, headers)
}
