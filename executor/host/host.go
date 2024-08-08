package host

import (
	"context"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/initia-labs/OPinit/x/ophost"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/keys"
	"github.com/initia-labs/opinit-bots-go/node"
	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
)

type childNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]btypes.ProcessedMsgs, bool) ([]types.RawKV, error)
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

	processedMsgs []btypes.ProcessedMsgs
	msgQueue      []sdk.Msg
}

func NewHost(
	version uint8, relayOracle bool, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix, batchSubmitter string,
) *Host {
	appCodec, txConfig, err := GetCodec(bech32Prefix)
	if err != nil {
		panic(err)
	}

	if batchSubmitter != "" {
		cfg.ProcessType = nodetypes.PROCESS_TYPE_ONLY_BROADCAST
		cfg.BroadcasterConfig.Bech32Prefix = bech32Prefix
		cfg.BroadcasterConfig.KeyringConfig.Address = batchSubmitter
	}

	node, err := node.NewNode(cfg, db, logger, appCodec, txConfig)
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

		ophostQueryClient: ophosttypes.NewQueryClient(node.GetRPCClient()),

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make([]sdk.Msg, 0),
	}

	return h
}

func GetCodec(bech32Prefix string) (codec.Codec, client.TxConfig, error) {
	unlock := keys.SetSDKConfigContext(bech32Prefix)
	defer unlock()

	return keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		ophost.AppModuleBasic{}.RegisterInterfaces,
	})
}

func (h *Host) Initialize(startHeight uint64, child childNode, batch batchNode, bridgeId int64) error {
	err := h.node.Initialize(startHeight)
	if err != nil {
		return err
	}
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
	h.logger.Info("host start", zap.Uint64("height", h.node.GetHeight()))
	h.node.Start(ctx)
}

func (h *Host) registerHandlers() {
	h.node.RegisterBeginBlockHandler(h.beginBlockHandler)
	h.node.RegisterTxHandler(h.txHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeFinalizeTokenWithdrawal, h.finalizeWithdrawalHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
	h.node.RegisterEventHandler(ophosttypes.EventTypeUpdateBatchInfo, h.updateBatchInfoHandler)
	h.node.RegisterEndBlockHandler(h.endBlockHandler)
}

func (h *Host) RegisterDAHandlers() {
	h.node.RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
}

func (h Host) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if len(msgs.Msgs) == 0 {
		return
	}

	h.node.MustGetBroadcaster().BroadcastMsgs(msgs)
}

func (h Host) ProcessedMsgsToRawKV(msgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	return h.node.MustGetBroadcaster().ProcessedMsgsToRawKV(msgs, delete)
}

func (h *Host) SetBridgeId(bridgeId int64) {
	h.bridgeId = bridgeId
}

func (h Host) BridgeId() int64 {
	return h.bridgeId
}

func (h Host) HasKey() bool {
	return h.node.HasBroadcaster()
}

func (h Host) GetHeight() uint64 {
	return h.node.GetHeight()
}
