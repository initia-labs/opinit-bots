package host

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/node"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/initia-labs/OPinit/x/ophost"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
)

type childNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(nodetypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]nodetypes.ProcessedMsgs, bool) ([]types.RawKV, error)
	QueryNextL1Sequence() (uint64, error)

	GetMsgFinalizeTokenDeposit(string, string, sdk.Coin, uint64, uint64, string, []byte) (sdk.Msg, error)
	GetMsgUpdateOracle(
		height uint64,
		data []byte,
	) (sdk.Msg, error)
}

type batchNode interface {
	UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber uint64)
}

var _ executortypes.DANode = &Host{}

type Host struct {
	version     uint8
	relayOracle bool

	node  *node.Node
	child childNode
	batch batchNode

	bridgeId          int64
	initialL1Sequence uint64

	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	ophostQueryClient ophosttypes.QueryClient

	processedMsgs []nodetypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(
	version uint8, relayOracle bool, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, homePath string, batchSubmitter string,
) *Host {
	appCodec, txConfig, bech32Prefix := GetCodec()
	processType := nodetypes.PROCESS_TYPE_DEFAULT
	if batchSubmitter != "" {
		processType = nodetypes.PROCESS_TYPE_ONLY_BROADCAST
	}

	node, err := node.NewNode(processType, cfg, db, logger, appCodec, txConfig, homePath, bech32Prefix, batchSubmitter)
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

		ophostQueryClient: ophosttypes.NewQueryClient(node),

		processedMsgs: make([]nodetypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return h
}

func GetCodec() (codec.Codec, client.TxConfig, string) {
	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec
	txConfig := encodingConfig.TxConfig

	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	auth.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ophost.AppModuleBasic{}.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return appCodec, txConfig, initiaapp.AccountAddressPrefix
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
	h.node.RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
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

func (h Host) HasKey() bool {
	return h.node.HasKey()
}

func (h Host) GetHeight() uint64 {
	return h.node.GetHeight()
}
