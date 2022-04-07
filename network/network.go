package network

import (
	"context"
	"io"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type (
	NewSealHandler             func(seal.Seal) error
	GetStagedOperationsHandler func([]valuehash.Hash) ([]operation.Operation, error)
	GetProposalHandler         func(valuehash.Hash) (base.Proposal, error)
	GetStateHandler            func(string) (state.State, bool, error)
	NodeInfoHandler            func() (NodeInfo, error)
	BlockdataMapsHandler       func([]base.Height) ([]block.BlockdataMap, error)
	BlockdataHandler           func(string) (io.Reader, func() error, error)
	StartHandoverHandler       func(StartHandoverSeal) (bool, error)
	PingHandoverHandler        func(PingHandoverSeal) (bool, error)
	EndHandoverHandler         func(EndHandoverSeal) (bool, error)
)

type Server interface {
	util.Daemon
	util.Initializer
	SetNewSealHandler(NewSealHandler)
	SetGetStagedOperationsHandler(GetStagedOperationsHandler)
	SetGetProposalHandler(GetProposalHandler)
	NodeInfoHandler() NodeInfoHandler
	SetNodeInfoHandler(NodeInfoHandler)
	SetBlockdataMapsHandler(BlockdataMapsHandler)
	SetBlockdataHandler(BlockdataHandler)
	SetStartHandoverHandler(StartHandoverHandler)
	SetPingHandoverHandler(PingHandoverHandler)
	SetEndHandoverHandler(EndHandoverHandler)
}

type Response interface {
	util.Byter
	OK() bool
}

var (
	ChannelTimeoutSeal         = time.Second * 7
	ChannelTimeoutOperation    = time.Second * 7
	ChannelTimeoutSendSeal     = time.Second * 7
	ChannelTimeoutNodeInfo     = time.Second * 7
	ChannelTimeoutBlockdataMap = time.Second * 7
	ChannelTimeoutBlockdata    = time.Minute
	ChannelTimeoutHandover     = time.Second * 7
)

type Channel interface {
	util.Initializer
	ConnInfo() ConnInfo
	StagedOperations(context.Context, []valuehash.Hash) ([]operation.Operation, error)
	SendSeal(context.Context, ConnInfo /* from ConnInfo */, seal.Seal) error
	Proposal(context.Context, valuehash.Hash) (base.Proposal, error)
	NodeInfo(context.Context) (NodeInfo, error)
	BlockdataMaps(context.Context, []base.Height) ([]block.BlockdataMap, error)
	Blockdata(context.Context, block.BlockdataMapItem) (io.ReadCloser, error)
	StartHandover(context.Context, StartHandoverSeal) (bool, error)
	PingHandover(context.Context, PingHandoverSeal) (bool, error)
	EndHandover(context.Context, EndHandoverSeal) (bool, error)
}
