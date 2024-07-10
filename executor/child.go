package executor

import (
	"context"

	"cosmossdk.io/core/address"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type hostNode interface {
	GetAddress() sdk.AccAddress
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
}

type child struct {
	node *node.Node
	host hostNode

	bridgeId int64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	cdc codec.Codec
	ac  address.Codec

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewChild(bridgeId int64, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, host hostNode) *child {
	node, err := node.NewNode(nodetypes.ChildNodeName, cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	ch := &child{
		node: node,
		host: host,

		bridgeId: bridgeId,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	ch.registerHandlers()
	return ch
}

func (ch *child) Start(ctx context.Context) {
	ch.node.Start(ctx)
}

func (ch *child) registerHostNode(host hostNode) {
	ch.host = host
}

func (ch *child) registerHandlers() {
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch child) GetAddress() sdk.AccAddress {
	return ch.node.GetAddress()
}

func (ch child) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	ch.node.BroadcastMsgs(msgs)
}

func (ch child) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return ch.node.RawKVProcessedData(msgs, delete)
}
