package host

import (
	"context"

	"cosmossdk.io/core/address"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
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
	QueryNextL1Sequence() (uint64, error)
}

type batchNode interface {
	UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber uint64)
}

var _ executortypes.DANode = &Host{}

type Host struct {
	version uint8

	node  *node.Node
	child childNode
	batch batchNode

	bridgeId          int64
	initialL1Sequence uint64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger
	cdc    codec.Codec
	ac     address.Codec

	ophostQueryClient ophosttypes.QueryClient

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(version uint8, cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Host {
	node, err := node.NewNode(cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	h := &Host{
		version: version,

		node: node,

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

func (h *Host) Initialize(child childNode, batch batchNode, bridgeId int64) (err error) {
	h.child = child
	h.batch = batch
	h.bridgeId = bridgeId

	h.initialL1Sequence, err = h.child.QueryNextL1Sequence()
	if err != nil {
		return err
	}

	h.registerHandlers()
	return nil
}

func (h *Host) Start(ctx context.Context) {
	h.node.Start(ctx, nodetypes.PROCESS_TYPE_DEFAULT)
}

func (h *Host) registerHandlers() {
	h.node.RegisterBeginBlockHandler(h.beginBlockHandler)
	h.node.RegisterTxHandler(h.txHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeFinalizeTokenWithdrawal, h.finalizeWithdrawalHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
	h.node.RegisterEndBlockHandler(h.endBlockHandler)
}

func (h Host) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if !h.node.HasKey() {
		return
	}
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
