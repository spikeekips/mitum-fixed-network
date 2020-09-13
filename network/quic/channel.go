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
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Channel struct {
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
	getStateURL     string
	nodeInfoURL     string
	client          *QuicClient
}

func NewChannel(
	addr string,
	bufsize uint,
	insecure bool,
	timeout time.Duration,
	retries int,
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

	if u, err := url.Parse(addr); err != nil {
		return nil, err
	} else {
		if u.Scheme == "quic" {
			u.Scheme = "https"
		}

		ch.addr = u
		ch.u = addr
	}

	ch.nodeInfoURL = mustQuicURL(ch.addr.String(), QuicHandlerPathNodeInfo)
	ch.sendSealURL = mustQuicURL(ch.addr.String(), QuicHandlerPathSendSeal)
	ch.getSealsURL = mustQuicURL(ch.addr.String(), QuicHandlerPathGetSeals)
	ch.getBlocksURL = mustQuicURL(ch.addr.String(), QuicHandlerPathGetBlocks)
	ch.getStateURL = mustQuicURL(ch.addr.String(), QuicHandlerPathGetState)
	ch.getManifestsURL = mustQuicURL(ch.addr.String(), QuicHandlerPathGetManifests)

	if client, err := NewQuicClient(insecure, timeout, retries, quicConfig); err != nil {
		return ch, nil
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

func (ch *Channel) Seals(hs []valuehash.Hash) ([]seal.Seal, error) {
	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range hs {
			l = append(l, h.String())
		}

		return e.Strs("seal_hashes", l)
	}).Msg("request seals")

	ss, err := ch.requestHinters(ch.getSealsURL, NewHashesArgs(hs))
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

func (ch *Channel) SendSeal(sl seal.Seal) error {
	b, err := ch.enc.Marshal(sl)
	if err != nil {
		return err
	}

	ch.Log().Debug().Hinted("seal_hash", sl.Hash()).Msg("sent seal")

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	return ch.client.Send(ch.sendSealURL, b, headers)
}

func (ch *Channel) requestHinters(u string, hs interface{}) ([]hint.Hinter, error) {
	b, err := ch.enc.Marshal(hs)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	response, err := ch.client.Request(u, b, headers)
	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header(), ch.encs, ch.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	var ss []json.RawMessage
	if err := enc.Unmarshal(response.Bytes(), &ss); err != nil {
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

func (ch *Channel) requestHinter(u string, hs interface{}) (hint.Hinter, error) {
	b, err := ch.enc.Marshal(hs)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	response, err := ch.client.Request(u, b, headers)
	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header(), ch.encs, ch.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	if hinter, err := enc.DecodeByHint(response.Bytes()); err != nil {
		return nil, err
	} else {
		return hinter, nil
	}
}

func (ch *Channel) Manifests(heights []base.Height) ([]block.Manifest, error) {
	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		return e.Strs("manifest_height", l)
	}).Msg("request manfests")

	hinters, err := ch.requestHinters(ch.getManifestsURL, NewHeightsArgs(heights))
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

func (ch *Channel) Blocks(heights []base.Height) ([]block.Block, error) {
	ch.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var l []string
		for _, h := range heights {
			l = append(l, h.String())
		}

		return e.Strs("block_heights", l)
	}).Msg("request blocks")

	hs, err := ch.requestHinters(ch.getBlocksURL, NewHeightsArgs(heights))
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

func (ch *Channel) State(key string) (state.State, bool, error) {
	ch.Log().Debug().Str("key", key).Msg("request state")

	if h, err := ch.requestHinter(ch.getStateURL, key); err != nil {
		if storage.IsNotFoundError(err) {
			return nil, false, nil
		}

		return nil, false, err
	} else if h == nil {
		return nil, false, nil
	} else if s, ok := h.(state.State); !ok {
		return nil, false, xerrors.Errorf("decoded, but not state.State; %T", h)
	} else {
		return s, true, nil
	}
}

func (ch *Channel) NodeInfo() (network.NodeInfo, error) {
	headers := http.Header{}
	headers.Set(QuicEncoderHintHeader, ch.enc.Hint().String())

	response, err := ch.client.Request(ch.nodeInfoURL, nil, headers)
	if err != nil {
		return nil, err
	} else if err := response.Error(); err != nil {
		return nil, err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(response.Header(), ch.encs, ch.enc); err != nil {
		return nil, err
	} else {
		enc = e
	}

	if hinter, err := network.DecodeNodeInfo(enc, response.Bytes()); err != nil {
		return nil, err
	} else {
		return hinter.(network.NodeInfo), nil
	}
}
