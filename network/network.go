package network

import (
	"context"
	"io"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type (
	GetSealsHandler      func([]valuehash.Hash) ([]seal.Seal, error)
	HasSealHandler       func(valuehash.Hash) (bool, error)
	NewSealHandler       func(seal.Seal) error
	GetStateHandler      func(string) (state.State, bool, error)
	NodeInfoHandler      func() (NodeInfo, error)
	BlockDataMapsHandler func([]base.Height) ([]block.BlockDataMap, error)
	BlockDataHandler     func(string) (io.Reader, func() error, error)
	StartHandoverHandler func(StartHandoverSeal) (bool, error)
	PingHandoverHandler  func(PingHandoverSeal) (bool, error)
	EndHandoverHandler   func(EndHandoverSeal) (bool, error)
)

type Server interface {
	util.Daemon
	util.Initializer
	SetHasSealHandler(HasSealHandler)
	SetGetSealsHandler(GetSealsHandler)
	SetNewSealHandler(NewSealHandler)
	NodeInfoHandler() NodeInfoHandler
	SetNodeInfoHandler(NodeInfoHandler)
	SetBlockDataMapsHandler(BlockDataMapsHandler)
	SetBlockDataHandler(BlockDataHandler)
	SetStartHandoverHandler(StartHandoverHandler)
	SetPingHandoverHandler(PingHandoverHandler)
	SetEndHandoverHandler(EndHandoverHandler)
}

type Response interface {
	util.Byter
	OK() bool
}

var (
	ChannelTimeoutSeal         = time.Second * 2
	ChannelTimeoutSendSeal     = time.Second * 2
	ChannelTimeoutNodeInfo     = time.Second * 2
	ChannelTimeoutBlockDataMap = time.Second * 2
	ChannelTimeoutBlockData    = time.Second * 30
	ChannelTimeoutHandover     = time.Second * 2
)

type Channel interface {
	util.Initializer
	ConnInfo() ConnInfo
	Seals(context.Context, []valuehash.Hash) ([]seal.Seal, error)
	SendSeal(context.Context, ConnInfo /* from ConnInfo */, seal.Seal) error
	NodeInfo(context.Context) (NodeInfo, error)
	BlockDataMaps(context.Context, []base.Height) ([]block.BlockDataMap, error)
	BlockData(context.Context, block.BlockDataMapItem) (io.ReadCloser, error)
	StartHandover(context.Context, StartHandoverSeal) (bool, error)
	PingHandover(context.Context, PingHandoverSeal) (bool, error)
	EndHandover(context.Context, EndHandoverSeal) (bool, error)
}
