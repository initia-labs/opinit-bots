package host

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
	GetAddressStr() (string, error)
	AccountCodec() address.Codec
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
}

type Host struct {
	version uint8

	node  *node.Node
	child childNode

	bridgeId int64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger
	cdc    codec.Codec
	ac     address.Codec

	ophostQueryClient ophosttypes.QueryClient

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(version uint8, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, child childNode) *Host {
	node, err := node.NewNode(cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	h := &Host{
		version: version,

		node:  node,
		child: child,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		ophostQueryClient: ophosttypes.NewQueryClient(node),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return h
}

func (h *Host) Start(ctx context.Context) {
	h.node.Start(ctx)
}

func (h *Host) RegisterHandlers() {
	h.node.RegisterBeginBlockHandler(h.beginBlockHandler)
	h.node.RegisterTxHandler(h.txHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.node.RegisterEndBlockHandler(h.endBlockHandler)
}

func (h Host) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	h.node.BroadcastMsgs(msgs)
}

func (h Host) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return h.node.RawKVProcessedData(msgs, delete)
}

func (h *Host) SetBridgeId(brigeId int64) {
	h.bridgeId = brigeId
}

func (h Host) AccountCodec() address.Codec {
	return h.ac
}

func (h Host) HasKey() bool {
	return h.node.HasKey()
}
