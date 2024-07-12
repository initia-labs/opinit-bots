package child

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/merkle"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots-go/node"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type hostNode interface {
	GetAddressStr() (string, error)
	AccountCodec() address.Codec
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	RawKVProcessedData([]nodetypes.ProcessedMsgs, bool) ([]types.KV, error)
	QueryLastOutput() (ophosttypes.QueryOutputProposalResponse, error)
}

type Child struct {
	version uint8

	node *node.Node
	host hostNode
	mk   *merkle.Merkle

	bridgeInfo opchildtypes.BridgeInfo

	nextOutputTime time.Time

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	cdc codec.Codec
	ac  address.Codec

	opchildQueryClient opchildtypes.QueryClient

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewChild(version uint8, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig, host hostNode) *Child {
	node, err := node.NewNode(cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	mk := merkle.NewMerkle(db.WithPrefix([]byte(executortypes.MerkleName)), ophosttypes.GenerateNodeHash)

	ch := &Child{
		version: version,

		node: node,
		host: host,
		mk:   mk,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		opchildQueryClient: opchildtypes.NewQueryClient(node),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return ch
}

func (ch *Child) Start(ctx context.Context) {
	ch.node.Start(ctx)
}

func (ch *Child) RegisterHandlers() {
	ch.node.RegisterBeginBlockHandler(ch.beginBlockHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeFinalizeTokenDeposit, ch.finalizeDepositHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeUpdateOracle, ch.updateOracleHandler)
	ch.node.RegisterEventHandler(opchildtypes.EventTypeInitiateTokenWithdrawal, ch.initiateWithdrawalHandler)
	ch.node.RegisterEndBlockHandler(ch.endBlockHandler)
}

func (ch Child) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	ch.node.BroadcastMsgs(msgs)
}

func (ch Child) RawKVProcessedData(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.KV, error) {
	return ch.node.RawKVProcessedData(msgs, delete)
}

func (ch Child) BridgeId() uint64 {
	return ch.bridgeInfo.BridgeId
}
func (ch Child) AccountCodec() address.Codec {
	return ch.ac
}

func (ch Child) HasKey() bool {
	return ch.node.HasKey()
}

func (ch *Child) SetBridgeInfo(bridgeInfo opchildtypes.BridgeInfo) {
	ch.bridgeInfo = bridgeInfo
}
