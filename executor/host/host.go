package host

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/cosmos/cosmos-sdk/codec"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	"github.com/pkg/errors"
)

var (
	BatchMsgType = "/opinit.ophost.v1.MsgRecordBatch"
)

type childNode interface {
	DB() types.DB
	Codec() codec.Codec

	HasBroadcaster() bool
	BroadcastProcessedMsgs(...btypes.ProcessedMsgs)

	GetMsgFinalizeTokenDeposit(string, string, sdk.Coin, uint64, int64, string, []byte) (sdk.Msg, string, error)
	GetMsgUpdateOracle(int64, []byte) (sdk.Msg, string, error)
	GetMsgSetBridgeInfo(uint64, ophosttypes.BridgeConfig) (sdk.Msg, string, error)

	QueryNextL1Sequence(context.Context, int64) (uint64, error)
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

	stage types.CommitDB

	// status info
	lastProposedOutputIndex         uint64
	lastProposedOutputL2BlockNumber int64
	lastProposedOutputTime          time.Time

	lastUpdatedBatchTime time.Time
}

func NewHostV1(cfg nodetypes.NodeConfig, db types.DB) *Host {
	return &Host{
		BaseHost: hostprovider.NewBaseHostV1(cfg, db),
		stage:    db.NewStage(),
	}
}

func (h *Host) Initialize(
	ctx types.Context,
	syncedHeight int64,
	child childNode,
	batch batchNode,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	keyringConfig *btypes.KeyringConfig,
) error {
	err := h.BaseHost.Initialize(ctx, syncedHeight, bridgeInfo, keyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize base host")
	}
	h.child = child
	h.batch = batch
	h.initialL1Sequence, err = h.child.QueryNextL1Sequence(ctx, 0)
	if err != nil {
		return errors.Wrap(err, "failed to query next L1 sequence")
	}
	h.registerHandlers()
	return h.LoadInternalStatus()
}

func (h *Host) InitializeDA(
	ctx types.Context,
	bridgeInfo ophosttypes.QueryBridgeResponse,
	keyringConfig *btypes.KeyringConfig,
) error {
	err := h.BaseHost.Initialize(ctx, 0, bridgeInfo, keyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize base DA host")
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
	h.Node().RegisterEventHandler(ophosttypes.EventTypeUpdateProposer, h.updateProposerHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeUpdateChallenger, h.updateChallengerHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeUpdateOracle, h.updateOracleConfigHandler)
	h.Node().RegisterEndBlockHandler(h.endBlockHandler)
}

func (h *Host) registerDAHandlers() {
	h.Node().RegisterEventHandler(ophosttypes.EventTypeRecordBatch, h.recordBatchHandler)
}

func (h *Host) LenProcessedBatchMsgs() (int, error) {
	broadcaster, err := h.Node().GetBroadcaster()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get broadcaster")
	}
	return broadcaster.LenProcessedMsgsByMsgType(BatchMsgType)
}

func (h *Host) LenPendingBatchTxs() (int, error) {
	broadcaster, err := h.Node().GetBroadcaster()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get broadcaster")
	}
	return broadcaster.LenLocalPendingTxByMsgType(BatchMsgType)
}

func (h Host) LastUpdatedBatchTime() time.Time {
	return h.lastUpdatedBatchTime
}
