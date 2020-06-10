package quicnetwork

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/lucas-clemente/quic-go"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

type QuicChannel struct {
	*logging.Logging
	recvChan        chan seal.Seal
	u               string
	addr            *url.URL
	encs            *encoder.Encoders
	enc             encoder.Encoder
	sendSealURL     string
	getSealsURL     string
	getManifestsURL string
	getBlocksURL    string
	nodeInfoURL     string
	client          *QuicClient
}

func NewQuicChannel(
	addr string,
	bufsize uint,
	insecure bool,
	timeout time.Duration,
	retries int,
	quicConfig *quic.Config,
	encs *encoder.Encoders,
	enc encoder.Encoder,
) (*QuicChannel, error) {
	qc := &QuicChannel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "quic-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
		encs:     encs,
		enc:      enc,
	}

	if u, err := url.Parse(addr); err != nil {
		return nil, err
	} else {
		if u.Scheme == "quic" {
			u.Scheme = "https"
		}

		qc.addr = u
		qc.u = addr
	}

	qc.nodeInfoURL = mustQuicURL(qc.addr.String(), QuicHandlerPathNodeInfo)
	qc.sendSealURL = mustQuicURL(qc.addr.String(), QuicHandlerPathSendSeal)
	qc.getSealsURL = mustQuicURL(qc.addr.String(), QuicHandlerPathGetSeals)
	qc.getBlocksURL = mustQuicURL(qc.addr.String(), QuicHandlerPathGetBlocks)
	qc.getManifestsURL = mustQuicURL(qc.addr.String(), QuicHandlerPathGetManifests)

	if client, err := NewQuicClient(insecure, timeout, retries, quicConfig); err != nil {
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

func (qc *QuicChannel) URL() string {
	return qc.u
}

func (qc *QuicChannel) Seals(hs []valuehash.Hash) ([]seal.Seal, error) {
	qc.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range hs {
			l = append(l, h.String())
		}

		return e.Strs("seal_hashes", l)
	}).Msg("request seals")

	ss, err := qc.requestHinters(qc.getSealsURL, NewHashesArgs(hs))
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

func (qc *QuicChannel) SendSeal(sl seal.Seal) error {
	b, err := qc.enc.Marshal(sl)
	if err != nil {
		return err
	}

	qc.Log().Debug().Hinted("seal_hash", sl.Hash()).Msg("sent seal")

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, qc.enc.Hint().String())

	return qc.client.Send(qc.sendSealURL, b, headers)
}

func (qc *QuicChannel) requestHinters(u string, hs interface{}) ([]hint.Hinter, error) {
	b, err := qc.enc.Marshal(hs)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, qc.enc.Hint().String())

	response, err := qc.client.Request(u, b, headers)
	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header(), qc.encs, qc.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	var ss []json.RawMessage
	if err := enc.Unmarshal(response.Bytes(), &ss); err != nil {
		qc.Log().Error().Err(err).Msg("failed to unmarshal manifest slice")
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

func (qc *QuicChannel) Manifests(heights []base.Height) ([]block.Manifest, error) {
	qc.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		return e.Strs("manifest_height", l)
	}).Msg("request manfests")

	hinters, err := qc.requestHinters(qc.getManifestsURL, NewHeightsArgs(heights))
	if err != nil {
		return nil, err
	}

	var manifests []block.Manifest
	for _, h := range hinters {
		if s, ok := h.(block.Manifest); !ok {
			return nil, xerrors.Errorf("decoded, but not Manifest; %T", h)
		} else {
			manifests = append(manifests, s)
		}
	}

	return manifests, nil
}

func (qc *QuicChannel) Blocks(heights []base.Height) ([]block.Block, error) {
	qc.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		return e.Strs("block_heights", l)
	}).Msg("request blocks")

	hs, err := qc.requestHinters(qc.getBlocksURL, NewHeightsArgs(heights))
	if err != nil {
		return nil, err
	}

	var blocks []block.Block
	for _, h := range hs {
		if s, ok := h.(block.Block); !ok {
			return nil, xerrors.Errorf("decoded, but not Block; %T", h)
		} else {
			blocks = append(blocks, s)
		}
	}

	return blocks, nil
}

func (qc *QuicChannel) NodeInfo() (network.NodeInfo, error) {
	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, qc.enc.Hint().String())

	response, err := qc.client.Request(qc.nodeInfoURL, nil, headers)
	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header(), qc.encs, qc.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	var ni network.NodeInfo
	if hinter, err := network.DecodeNodeInfo(enc, response.Bytes()); err != nil {
		return nil, err
	} else {
		ni = hinter.(network.NodeInfo)
	}

	return ni, nil
}
