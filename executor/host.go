package executor

import (
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
	*Executor

	node  *node.Node
	child childNode

	bridgeId int64

	cfg nodetypes.NodeConfig
	cdc codec.Codec
	ac  address.Codec

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(executor *Executor, bridgeId int64, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *host {
	node, err := node.NewNode(nodetypes.HostNodeName, cfg, db.WithPrefix([]byte(nodetypes.HostNodeName)), logger.Named(nodetypes.HostNodeName), cdc, txConfig)
	if err != nil {
		panic(err)
	}

	return &host{
		Executor: executor,

		node: node,

		bridgeId: bridgeId,

		cfg: cfg,
		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}
}

func (h *host) registerChildNode(node childNode) {
	h.child = node
}

func (h host) registerHostHandlers() {
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
