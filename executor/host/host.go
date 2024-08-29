package host

import (
	"context"

	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
	QueryNextL1Sequence(context.Context, uint64) (uint64, error)

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
	*hostprovider.BaseHost

	child childNode
	batch batchNode

	relayOracle       bool
	initialL1Sequence uint64

	// status info
	lastProposedOutputIndex         uint64
	lastProposedOutputL2BlockNumber uint64
}

func NewHostV0(
	relayOracle bool, cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix, batchSubmitter string,
) *Host {
	if batchSubmitter != "" {
		cfg.BroadcasterConfig.Bech32Prefix = bech32Prefix
		cfg.BroadcasterConfig.KeyringConfig.Address = batchSubmitter
	}
	return &Host{
		relayOracle: relayOracle,
		BaseHost:    hostprovider.NewBaseHostV0(cfg, db, logger, bech32Prefix),
	}
}

func (h *Host) Initialize(ctx context.Context, startHeight uint64, child childNode, batch batchNode, bridgeId int64) error {
	err := h.BaseHost.Initialize(ctx, startHeight, bridgeId)
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

func (h *Host) RegisterDAHandlers() {
	h.Node().RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
}
