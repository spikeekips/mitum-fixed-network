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
	BlockDataHandler     func(string) (io.ReadCloser, func() error, error)
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
)

type Channel interface {
	util.Initializer
	URL() string
	Seals(context.Context, []valuehash.Hash) ([]seal.Seal, error)
	SendSeal(context.Context, seal.Seal) error
	NodeInfo(context.Context) (NodeInfo, error)
	BlockDataMaps(context.Context, []base.Height) ([]block.BlockDataMap, error)
	BlockData(context.Context, block.BlockDataMapItem) (io.ReadCloser, error)
}
