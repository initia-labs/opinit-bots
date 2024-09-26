package host

import (
	"context"

	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
)

type childNode interface {
	GetAddressStr() (string, error)
	HasKey() bool
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV([]btypes.ProcessedMsgs, bool) ([]types.RawKV, error)
	QueryNextL1Sequence(context.Context, int64) (uint64, error)

	GetMsgFinalizeTokenDeposit(string, string, sdk.Coin, uint64, int64, string, []byte) (sdk.Msg, error)
	GetMsgUpdateOracle(int64, []byte) (sdk.Msg, error)
}

type batchNode interface {
	UpdateBatchInfo(string, string, uint64, int64)
}

var _ executortypes.DANode = &Host{}

type Host struct {
	*hostprovider.BaseHost

	child childNode
	batch batchNode

	initialL1Sequence uint64

	// status info
	lastProposedOutputIndex         uint64
	lastProposedOutputL2BlockNumber int64
}

func NewHostV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix, batchSubmitter string,
) *Host {
	if cfg.BroadcasterConfig != nil && batchSubmitter != "" {
		cfg.BroadcasterConfig.Bech32Prefix = bech32Prefix
		cfg.BroadcasterConfig.KeyringConfig.Address = batchSubmitter
	}
	return &Host{
		BaseHost: hostprovider.NewBaseHostV1(cfg, db, logger, bech32Prefix),
	}
}

func (h *Host) Initialize(ctx context.Context, startHeight int64, child childNode, batch batchNode, bridgeInfo opchildtypes.BridgeInfo) error {
	err := h.BaseHost.Initialize(ctx, startHeight, bridgeInfo)
	if err != nil {
		return err
	}
	h.child = child
	h.batch = batch
	h.initialL1Sequence, err = h.child.QueryNextL1Sequence(ctx, 0)
	if err != nil {
		return err
	}
	h.registerHandlers()
	return nil
}

func (h *Host) InitializeDA(ctx context.Context, bridgeInfo opchildtypes.BridgeInfo) error {
	err := h.BaseHost.Initialize(ctx, 0, bridgeInfo)
	if err != nil {
		return err
	}
	h.registerDAHandlers()
	return nil
}

func (h *Host) registerHandlers() {
	h.Node().RegisterBeginBlockHandler(h.beginBlockHandler)
	h.Node().RegisterTxHandler(h.txHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeFinalizeTokenWithdrawal, h.finalizeWithdrawalHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeUpdateBatchInfo, h.updateBatchInfoHandler)
	h.Node().RegisterEndBlockHandler(h.endBlockHandler)
}

func (h *Host) registerDAHandlers() {
	h.Node().RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
}
