package host

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/node"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	slinkycodec "github.com/skip-mev/slinky/abci/strategies/codec"
)

type childNode interface {
	GetAddressStr() (string, error)
	AccountCodec() address.Codec
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]nodetypes.ProcessedMsgs, bool) ([]types.RawKV, error)
	QueryNextL1Sequence() (uint64, error)
}

type Host struct {
	version     uint8
	relayOracle bool

	node  *node.Node
	child childNode

	bridgeId          int64
	initialL1Sequence uint64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger
	cdc    codec.Codec
	ac     address.Codec

	ophostQueryClient   ophosttypes.QueryClient
	extendedCommitCodec slinkycodec.ExtendedCommitCodec

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(
	version uint8, relayOracle bool, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig,
) *Host {
	node, err := node.NewNode(cfg, db, logger, cdc, txConfig)
	if err != nil {
		panic(err)
	}

	h := &Host{
		version:     version,
		relayOracle: relayOracle,

		node: node,

		cfg:    cfg,
		db:     db,
		logger: logger,

		cdc: cdc,
		ac:  cdc.InterfaceRegistry().SigningContext().AddressCodec(),

		ophostQueryClient: ophosttypes.NewQueryClient(node),
		extendedCommitCodec: slinkycodec.NewCompressionExtendedCommitCodec(
			slinkycodec.NewDefaultExtendedCommitCodec(),
			slinkycodec.NewZStdCompressor(),
		),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return h
}

func (h *Host) Initialize(child childNode, bridgeId int64) (err error) {
	h.child = child
	h.bridgeId = bridgeId

	h.initialL1Sequence, err = h.child.QueryNextL1Sequence()
	if err != nil {
		return err
	}

	h.registerHandlers()

	return nil
}

func (h *Host) Start(ctx context.Context, errCh chan error) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("host panic", zap.Any("recover", r))
			errCh <- fmt.Errorf("host panic: %v", r)
		}
	}()

	h.node.Start(ctx, errCh)
}

func (h *Host) registerHandlers() {
	h.node.RegisterBeginBlockHandler(h.beginBlockHandler)
	h.node.RegisterTxHandler(h.txHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeFinalizeTokenWithdrawal, h.finalizeWithdrawalHandler)
	h.node.RegisterEndBlockHandler(h.endBlockHandler)
}

func (h Host) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if !h.node.HasKey() {
		return
	}

	h.node.BroadcastMsgs(msgs)
}

func (h Host) ProcessedMsgsToRawKV(msgs []nodetypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	return h.node.ProcessedMsgsToRawKV(msgs, delete)
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

func (ch Host) GetHeight() uint64 {
	return ch.node.GetHeight()
}
