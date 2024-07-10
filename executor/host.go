package executor

import (
	"context"

	"cosmossdk.io/core/address"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type childNode interface {
	GetAddress() sdk.AccAddress
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
}

type host struct {
	node  *node.Node
	child childNode

	bridgeId int64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger
	cdc    codec.Codec
	ac     address.Codec

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(bridgeId int64, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, child childNode) *host {
	node, err := node.NewNode(nodetypes.HostNodeName, cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	h := &host{
		node:  node,
		child: child,

		bridgeId: bridgeId,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	h.registerHandlers()
	return h
}

func (h *host) Start(ctx context.Context) {
	h.node.Start(ctx)
}

func (h *host) registerChildNode(child childNode) {
	h.child = child
}

func (h host) registerHandlers() {
	h.node.RegisterBeginBlockHandler(h.beginBlockHandler)
	h.node.RegisterTxHandler(h.txHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.node.RegisterEndBlockHandler(h.endBlockHandler)
}

func (h host) GetAddress() sdk.AccAddress {
	return h.node.GetAddress()
}

func (h host) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	h.node.BroadcastMsgs(msgs)
}

func (h host) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return h.node.RawKVProcessedData(msgs, delete)
}
